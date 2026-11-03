"""The Windows app: a click-through overlay over every monitor that the snow is drawn into, a watcher that tells
the simulation where the windows are, and the tray icon with its menu."""

from __future__ import annotations

import ctypes
import logging
import os
import sys
import time
import traceback
from ctypes import wintypes as wt

import numpy as np

from . import prefs as prefs_mod
from . import strings
from . import win32 as w
from .icon import bgra_premultiplied
from .render import Canvas, draw
from .sim import Rect, Settings, SurfaceSpec, World

RUN_NAME = "SnowForWindows"
TRAY_MSG = w.WM_APP + 1
MENU_TIMER = 1

CMD_LEVEL = 100  # followed by one id per snowfall level
CMD_ACCUMULATE, CMD_MELT, CMD_ON_WINDOWS, CMD_CLEAR = 200, 201, 202, 203
CMD_SCREENSHOTS, CMD_SMOOTH, CMD_LOGIN, CMD_ABOUT, CMD_QUIT = 204, 205, 206, 207, 208

# Shell windows that never hold snow: the desktop, the taskbar, shadows and tooltips.
IGNORED = {
    "Progman", "WorkerW", "Shell_TrayWnd", "Shell_SecondaryTrayWnd", "tooltips_class32", "SysShadow",
    "EdgeUiInputTopWndClass", "EdgeUiInputWndClass", "ApplicationManager_ImmersiveShellWindow",
    "Internet Explorer_Hidden", "SnowForWindowsOverlay", "SnowForWindowsTray",
}
# Switchers, menus and shell flyouts hide the snow behind them but don't collect any.
COVER_ONLY = {
    "XamlExplorerHostIslandWindow", "MultitaskingViewFrame", "TaskListThumbnailWnd", "ForegroundStaging",
    "TopLevelWindowForOverflowXamlIsland", "NotifyIconOverflowWindow", "Xaml_WindowedPopupClass",
    "Windows.UI.Core.CoreWindow", "#32768",
}

log = logging.getLogger("snow")


def _rect(r: wt.RECT) -> tuple[int, int, int, int]:
    return r.left, r.top, r.right, r.bottom


class App:
    def __init__(self) -> None:
        self.text = strings.for_language(w.GetUserDefaultUILanguage())
        self.prefs = prefs_mod.load()
        self.inst = w.GetModuleHandleW(None)
        self.world = World(1, 1, 1.0)
        self.canvas: Canvas | None = None
        self.overlay = self.tray = None
        self.icon = None
        self.mem_dc = self.bmp = self.old_bmp = None
        self.vx = self.vy = self.vw = self.vh = 0
        self.scale = 0.0
        self.monitors: list[tuple[tuple, tuple]] = []  # (full, work) rectangles
        self.windows: list[tuple[int, tuple, bool]] = []  # (hwnd, frame, collects snow)
        self.shown = False
        self.locked = False
        self.busy = False
        self.layout_dirty = False
        self.menu_open = False
        self.last = 0.0
        self.next_track = self.next_busy = self.next_top = self.next_monitors = 0.0
        # ctypes callbacks must stay referenced for as long as Windows may call them.
        self._overlay_proc = w.WNDPROC(self._safe(self._overlay_wndproc, default=0))
        self._tray_proc = w.WNDPROC(self._safe(self._tray_wndproc, default=0))
        self._enum_windows = w.WNDENUMPROC(self._safe(self._on_window, default=1))
        self._enum_monitors = w.MONITORENUMPROC(self._safe(self._on_monitor, default=1))
        self.taskbar_created = 0

    # ---- setup -----------------------------------------------------------------------------------------------

    @staticmethod
    def _safe(fn, default):
        """Keep an exception in a callback from vanishing silently inside Windows."""

        def wrapper(*args):
            try:
                return fn(*args)
            except Exception:
                log.error("callback failed\n%s", traceback.format_exc())
                return default

        return wrapper

    def create_windows(self) -> bool:
        for name, proc in (("SnowForWindowsOverlay", self._overlay_proc), ("SnowForWindowsTray", self._tray_proc)):
            wc = w.WNDCLASSEXW()
            wc.cbSize = ctypes.sizeof(wc)
            wc.lpfnWndProc = proc
            wc.hInstance = self.inst
            wc.hCursor = w.LoadCursorW(None, ctypes.c_void_p(32512))  # IDC_ARROW
            wc.lpszClassName = name
            if not w.RegisterClassExW(ctypes.byref(wc)):
                log.error("RegisterClassExW %s failed: %d", name, ctypes.get_last_error())
                return False
        ex = w.WS_EX_LAYERED | w.WS_EX_TRANSPARENT | w.WS_EX_TOOLWINDOW | w.WS_EX_NOACTIVATE | w.WS_EX_TOPMOST
        self.overlay = w.CreateWindowExW(ex, "SnowForWindowsOverlay", "Snow", w.WS_POPUP, 0, 0, 1, 1,
                                         None, None, self.inst, None)
        self.tray = w.CreateWindowExW(w.WS_EX_TOOLWINDOW, "SnowForWindowsTray", "Snow for Windows", w.WS_POPUP,
                                      0, 0, 0, 0, None, None, self.inst, None)
        if not self.overlay or not self.tray:
            log.error("CreateWindowExW failed: %d", ctypes.get_last_error())
            return False
        self.taskbar_created = w.RegisterWindowMessageW("TaskbarCreated")
        w.WTSRegisterSessionNotification(self.overlay, 0)  # NOTIFY_FOR_THIS_SESSION
        size = max(16, w.GetSystemMetrics(w.SM_CXSMICON))
        self.icon = w.create_icon(bgra_premultiplied(size), size)
        return True

    # ---- window procedures -----------------------------------------------------------------------------------

    def _overlay_wndproc(self, hwnd, msg, wp, lp):
        if msg == w.WM_NCHITTEST:
            return w.HTTRANSPARENT
        if msg in (w.WM_DISPLAYCHANGE, w.WM_DPICHANGED, w.WM_SETTINGCHANGE):
            self.layout_dirty = True
            return 0
        if msg == w.WM_WTSSESSION_CHANGE:
            if wp == w.WTS_SESSION_LOCK:
                self.locked = True
            elif wp == w.WTS_SESSION_UNLOCK:
                self.locked = False
            return 0
        return w.DefWindowProcW(hwnd, msg, wp, lp)

    def _tray_wndproc(self, hwnd, msg, wp, lp):
        if msg == TRAY_MSG:
            if lp & 0xFFFF in (w.WM_LBUTTONUP, w.WM_RBUTTONUP):
                self.show_menu()
            return 0
        if msg == w.WM_TIMER and wp == MENU_TIMER:
            # The menu runs its own message loop; keep the snow moving while it is open.
            self.frame()
            return 0
        if self.taskbar_created and msg == self.taskbar_created:
            self.add_tray()  # Explorer restarted, and its tray is new and empty
            return 0
        if msg in (w.WM_DISPLAYCHANGE, w.WM_SETTINGCHANGE):
            self.layout_dirty = True
        return w.DefWindowProcW(hwnd, msg, wp, lp)

    # ---- layout ----------------------------------------------------------------------------------------------

    def rebuild(self) -> None:
        """Size the overlay to the virtual screen and find the ground along each monitor."""
        self.layout_dirty = False
        x, y, width, height = w.virtual_screen()
        if width <= 0 or height <= 0:
            return
        scale = w.system_scale()
        if (width, height, scale) != (self.vw, self.vh, self.scale) or not self.mem_dc:
            self._free_bitmap()
            screen = w.GetDC(None)
            self.mem_dc = w.CreateCompatibleDC(screen)
            w.ReleaseDC(None, screen)
            self.bmp, bits = w.create_dib(width, height)
            if not self.bmp:
                log.error("CreateDIBSection %dx%d failed", width, height)
                self.canvas = None
                return
            self.old_bmp = w.SelectObject(self.mem_dc, self.bmp)
            pixels = np.ctypeslib.as_array((ctypes.c_uint32 * (width * height)).from_address(bits))
            self.canvas = Canvas(pixels.reshape(height, width))
            self.world.resize(width, height, scale)
            self.vw, self.vh, self.scale = width, height, scale
            log.info("screen %dx%d at (%d, %d), scale %.2f", width, height, x, y, scale)
        self.vx, self.vy = x, y
        self.monitors = self._read_monitors()
        self.canvas.invalidate()
        self.next_track = 0.0

    def _read_monitors(self):
        found = []
        self._found_monitors = found
        w.EnumDisplayMonitors(None, None, self._enum_monitors, 0)
        return found

    def _on_monitor(self, hmon, hdc, rc, lp):
        mi = w.MONITORINFO()
        mi.cbSize = ctypes.sizeof(mi)
        if w.GetMonitorInfoW(hmon, ctypes.byref(mi)):
            self._found_monitors.append((_rect(mi.rcMonitor), _rect(mi.rcWork)))
        return 1

    def check_monitors(self) -> None:
        """Rebuild if the screens changed without a message saying so."""
        x, y, width, height = w.virtual_screen()
        if ((x, y, width, height) != (self.vx, self.vy, self.vw, self.vh) or w.system_scale() != self.scale
                or self._read_monitors() != self.monitors):
            self.rebuild()

    def _free_bitmap(self) -> None:
        if self.mem_dc:
            w.SelectObject(self.mem_dc, self.old_bmp)
            w.DeleteDC(self.mem_dc)
            self.mem_dc = None
        if self.bmp:
            w.DeleteObject(self.bmp)
            self.bmp = None

    def _on_window(self, hwnd, lp):
        if hwnd in (self.overlay, self.tray) or not w.IsWindowVisible(hwnd) or w.IsIconic(hwnd) or w.is_cloaked(hwnd):
            return 1
        cls = w.class_name(hwnd)
        if cls in IGNORED:
            return 1
        ex = w.GetWindowLongPtrW(hwnd, w.GWL_EXSTYLE)
        if ex & w.WS_EX_LAYERED and ex & w.WS_EX_TRANSPARENT:
            return 1  # another click-through overlay
        frame = w.window_frame(hwnd)
        if frame is None or frame.right - frame.left < 2 or frame.bottom - frame.top < 2:
            return 1
        l, t, r, b = _rect(frame)
        sc = self.scale
        collects = (cls not in COVER_ONLY and not ex & w.WS_EX_TOOLWINDOW and not w.IsZoomed(hwnd)
                    and r - l >= 120 * sc and b - t >= 50 * sc)
        if collects:
            # No snow on a window whose top is at the top of its monitor, as a maximized or snapped one's is.
            cx = (l + r) // 2
            for full, work in self.monitors:
                if full[0] <= cx < full[2] and full[1] <= t < full[3]:
                    collects = t > work[1] + 4
                    break
        self.windows.append((hwnd, (l, t, r, b), collects))
        return 1

    def track(self) -> None:
        """Tell the simulation where the windows and the ground are. Windows come front first."""
        self.windows = []
        w.EnumWindows(self._enum_windows, 0)
        occluders, specs = [], []
        inset = int(3 * self.scale)  # stay clear of rounded corners
        vx, vy = self.vx, self.vy
        for i, (hwnd, (l, t, r, b), collects) in enumerate(self.windows):
            occluders.append(Rect(l - vx, t - vy, r - vx, b - vy))
            if collects and self.prefs.snow_on_windows:
                specs.append(SurfaceSpec(int(hwnd), l - vx + inset, r - vx - inset, t - vy, z=i))
        for i, (full, work) in enumerate(self.monitors):
            specs.append(SurfaceSpec((1 << 63) | i, work[0] - vx, work[2] - vx, work[3] - vy, ground=True,
                                     z=len(occluders)))
        self.world.set_layout(specs, occluders)

    # ---- drawing ---------------------------------------------------------------------------------------------

    def show(self) -> None:
        if not self.shown:
            w.ShowWindow(self.overlay, w.SW_SHOWNOACTIVATE)
            self._raise()
            self.shown = True

    def hide(self) -> None:
        if self.shown:
            w.ShowWindow(self.overlay, w.SW_HIDE)
            self.shown = False
            if self.canvas:
                self.canvas.invalidate()

    def _raise(self) -> None:
        w.SetWindowPos(self.overlay, w.HWND_TOPMOST, 0, 0, 0, 0, w.SWP_NOMOVE | w.SWP_NOSIZE | w.SWP_NOACTIVATE)

    def frame_interval(self) -> float:
        return 1 / 60 if self.prefs.smooth_60fps else 1 / 30

    def frame(self) -> float:
        """Advance and draw one frame; returns how long the loop may sleep before the next one."""
        now = time.perf_counter()
        dt = now - self.last if self.last else 0.0
        self.last = now
        if self.layout_dirty or now >= self.next_monitors:
            if self.layout_dirty:
                self.rebuild()
            else:
                self.check_monitors()
            self.next_monitors = now + 3
        if now >= self.next_busy:
            self.busy = w.shell_busy()
            self.next_busy = now + 0.5
        if self.locked or self.busy or self.canvas is None:
            self.hide()
            self.last = 0.0
            return 0.5
        if now >= self.next_track:
            self.track()
            self.next_track = now + 0.08
        pt = wt.POINT()
        if w.GetCursorPos(ctypes.byref(pt)):
            self.world.set_cursor(pt.x - self.vx, pt.y - self.vy, True, dt)
        else:
            self.world.set_cursor(0, 0, False, dt)
        self.world.update(dt)
        if self.world.idle():
            self.hide()
            self.last = 0.0
            return 1.0
        draw(self.world, self.canvas)
        w.update_layered(self.overlay, self.mem_dc, self.vx, self.vy, self.vw, self.vh)
        self.show()
        if now >= self.next_top:
            self._raise()  # other topmost windows come and go; stay in front of them
            self.next_top = now + 2
        return max(self.frame_interval() - (time.perf_counter() - now), 0.001)

    def loop(self) -> None:
        timer = w.new_frame_timer()
        msg = wt.MSG()
        while True:
            while w.PeekMessageW(ctypes.byref(msg), None, 0, 0, w.PM_REMOVE):
                if msg.message == w.WM_QUIT:
                    return
                w.TranslateMessage(ctypes.byref(msg))
                w.DispatchMessageW(ctypes.byref(msg))
            try:
                pause = self.frame()
            except Exception:
                log.error("frame failed\n%s", traceback.format_exc())
                pause = 1.0
            w.wait(timer, pause)

    # ---- settings and tray -----------------------------------------------------------------------------------

    def apply(self, save: bool) -> None:
        amount, variable = prefs_mod.level(self.prefs.intensity)
        self.world.settings = Settings(amount, variable, self.prefs.accumulate, self.prefs.melt)
        w.exclude_from_capture(self.overlay, not self.prefs.show_in_screenshots)
        self.next_track = 0.0
        if save:
            prefs_mod.save(self.prefs)
            self.update_tray()

    def _tray_data(self) -> w.NOTIFYICONDATAW:
        d = w.NOTIFYICONDATAW()
        d.cbSize = ctypes.sizeof(d)
        d.hWnd = self.tray
        d.uID = 1
        d.uFlags = w.NIF_MESSAGE | w.NIF_ICON | w.NIF_TIP
        d.uCallbackMessage = TRAY_MSG
        d.hIcon = self.icon
        tip = self.text.paused_tip if self.prefs.intensity == prefs_mod.OFF else self.text.tip
        d.szTip = tip[:127]
        return d

    def add_tray(self) -> None:
        w.Shell_NotifyIconW(w.NIM_ADD, ctypes.byref(self._tray_data()))

    def update_tray(self) -> None:
        w.Shell_NotifyIconW(w.NIM_MODIFY, ctypes.byref(self._tray_data()))

    def remove_tray(self) -> None:
        d = w.NOTIFYICONDATAW()
        d.cbSize = ctypes.sizeof(d)
        d.hWnd = self.tray
        d.uID = 1
        w.Shell_NotifyIconW(w.NIM_DELETE, ctypes.byref(d))

    def balloon(self, message: str) -> None:
        d = self._tray_data()
        d.uFlags |= w.NIF_INFO
        d.dwInfoFlags = w.NIIF_USER | w.NIIF_LARGE_ICON
        d.hBalloonIcon = self.icon
        d.szInfo = message[:255]
        d.szInfoTitle = self.text.app_name[:63]
        w.Shell_NotifyIconW(w.NIM_MODIFY, ctypes.byref(d))

    def show_menu(self) -> None:
        if self.menu_open:
            return
        self.menu_open = True
        cmd = 0
        try:
            t, p = self.text, self.prefs

            def check(on: bool) -> int:
                return w.MF_STRING | (w.MF_CHECKED if on else 0)

            menu = w.CreatePopupMenu()
            levels = w.CreatePopupMenu()
            for i, name in enumerate(t.levels):
                w.AppendMenuW(levels, check(p.intensity == i), CMD_LEVEL + i, name)
            w.AppendMenuW(menu, w.MF_POPUP, levels, t.snowfall)
            w.AppendMenuW(menu, w.MF_SEPARATOR, 0, None)
            w.AppendMenuW(menu, check(p.accumulate), CMD_ACCUMULATE, t.accumulate)
            w.AppendMenuW(menu, check(p.melt), CMD_MELT, t.melt)
            w.AppendMenuW(menu, check(p.snow_on_windows), CMD_ON_WINDOWS, t.on_windows)
            w.AppendMenuW(menu, w.MF_STRING, CMD_CLEAR, t.clear)
            w.AppendMenuW(menu, w.MF_SEPARATOR, 0, None)
            w.AppendMenuW(menu, check(p.show_in_screenshots), CMD_SCREENSHOTS, t.in_screenshots)
            w.AppendMenuW(menu, check(p.smooth_60fps), CMD_SMOOTH, t.smooth)
            w.AppendMenuW(menu, check(w.run_at_login(RUN_NAME)), CMD_LOGIN, t.start_at_login)
            w.AppendMenuW(menu, w.MF_SEPARATOR, 0, None)
            w.AppendMenuW(menu, w.MF_STRING, CMD_ABOUT, t.about)
            w.AppendMenuW(menu, w.MF_STRING, CMD_QUIT, t.quit)

            pt = wt.POINT()
            w.GetCursorPos(ctypes.byref(pt))
            w.SetForegroundWindow(self.tray)
            w.SetTimer(self.tray, MENU_TIMER, int(self.frame_interval() * 1000), None)
            cmd = w.TrackPopupMenu(menu, w.TPM_RIGHTBUTTON | w.TPM_RETURNCMD | w.TPM_BOTTOMALIGN, pt.x, pt.y, 0,
                                   self.tray, None)
            w.KillTimer(self.tray, MENU_TIMER)
            w.PostMessageW(self.tray, w.WM_NULL, 0, 0)
            w.DestroyMenu(menu)
        finally:
            self.menu_open = False
        self.command(cmd)

    def command(self, cmd: int) -> None:
        p = self.prefs
        if CMD_LEVEL <= cmd <= CMD_LEVEL + prefs_mod.VARIABLE:
            p.intensity = cmd - CMD_LEVEL
        elif cmd == CMD_ACCUMULATE:
            p.accumulate = not p.accumulate
        elif cmd == CMD_MELT:
            p.melt = not p.melt
        elif cmd == CMD_ON_WINDOWS:
            p.snow_on_windows = not p.snow_on_windows
        elif cmd == CMD_SCREENSHOTS:
            p.show_in_screenshots = not p.show_in_screenshots
        elif cmd == CMD_SMOOTH:
            p.smooth_60fps = not p.smooth_60fps
        elif cmd == CMD_LOGIN:
            w.set_run_at_login(RUN_NAME, launch_command(), not w.run_at_login(RUN_NAME))
            return
        elif cmd == CMD_CLEAR:
            self.world.clear()
            return
        elif cmd == CMD_ABOUT:
            w.MessageBoxW(self.tray, self.text.about_text, self.text.app_name, w.MB_ICONINFORMATION)
            return
        elif cmd == CMD_QUIT:
            w.PostQuitMessage(0)
            return
        else:
            return
        self.apply(save=True)


def launch_command() -> str:
    """What the Run key should start: the exe when frozen by PyInstaller, else pythonw with this package."""
    if getattr(sys, "frozen", False):
        return f'"{sys.executable}"'
    exe = sys.executable
    pythonw = os.path.join(os.path.dirname(exe), "pythonw.exe")
    if os.path.exists(pythonw):
        exe = pythonw
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    return f'"{exe}" "{os.path.join(root, "run.pyw")}"'


def _setup_logging() -> None:
    try:
        os.makedirs(prefs_mod.folder(), exist_ok=True)
        logging.basicConfig(filename=os.path.join(prefs_mod.folder(), "log.txt"), filemode="w", level=logging.INFO,
                            format="%(asctime)s %(levelname)s %(message)s")
    except OSError:
        logging.basicConfig(level=logging.INFO)


def main() -> None:
    _setup_logging()
    w.set_dpi_aware()
    if not w.single_instance(r"Local\SnowForWindows-7c1d2f"):
        return
    app = App()
    if not app.create_windows():
        w.MessageBoxW(None, "Could not create the snow window.", app.text.app_name, 0)
        return
    app.rebuild()
    app.apply(save=False)
    app.add_tray()
    if not app.prefs.welcomed:
        app.balloon(app.text.welcome)
        app.prefs.welcomed = True
        prefs_mod.save(app.prefs)
    log.info("running")
    try:
        app.loop()
    finally:
        app.remove_tray()
        w.ShowWindow(app.overlay, w.SW_HIDE)
        log.info("quit")
