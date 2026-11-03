"""The Win32 calls the app needs, through ctypes, with their argument types spelled out so 64-bit handles and
negative coordinates pass through intact."""

from __future__ import annotations

import ctypes
from ctypes import wintypes as wt

user32 = ctypes.WinDLL("user32", use_last_error=True)
gdi32 = ctypes.WinDLL("gdi32", use_last_error=True)
kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
shell32 = ctypes.WinDLL("shell32", use_last_error=True)
dwmapi = ctypes.WinDLL("dwmapi", use_last_error=True)
advapi32 = ctypes.WinDLL("advapi32", use_last_error=True)
wtsapi32 = ctypes.WinDLL("wtsapi32", use_last_error=True)

LRESULT = ctypes.c_ssize_t
LONG_PTR = ctypes.c_ssize_t
UINT_PTR = ctypes.c_size_t
HCURSOR = wt.HANDLE

WNDPROC = ctypes.WINFUNCTYPE(LRESULT, wt.HWND, wt.UINT, wt.WPARAM, wt.LPARAM)
WNDENUMPROC = ctypes.WINFUNCTYPE(wt.BOOL, wt.HWND, wt.LPARAM)
MONITORENUMPROC = ctypes.WINFUNCTYPE(wt.BOOL, wt.HMONITOR, wt.HDC, ctypes.POINTER(wt.RECT), wt.LPARAM)

# Messages and flags.
WM_NULL = 0x0000
WM_SETTINGCHANGE = 0x001A
WM_DISPLAYCHANGE = 0x007E
WM_NCHITTEST = 0x0084
WM_TIMER = 0x0113
WM_QUIT = 0x0012
WM_LBUTTONUP = 0x0202
WM_RBUTTONUP = 0x0205
WM_WTSSESSION_CHANGE = 0x02B1
WM_DPICHANGED = 0x02E0
WM_APP = 0x8000
WTS_SESSION_LOCK = 0x7
WTS_SESSION_UNLOCK = 0x8
HTTRANSPARENT = -1

WS_POPUP = 0x80000000
WS_EX_TOPMOST = 0x00000008
WS_EX_TRANSPARENT = 0x00000020
WS_EX_TOOLWINDOW = 0x00000080
WS_EX_LAYERED = 0x00080000
WS_EX_NOACTIVATE = 0x08000000
GWL_EXSTYLE = -20

SW_HIDE = 0
SW_SHOWNOACTIVATE = 4
SWP_NOSIZE = 0x0001
SWP_NOMOVE = 0x0002
SWP_NOACTIVATE = 0x0010
HWND_TOPMOST = wt.HWND(-1)
HWND_NOTOPMOST = wt.HWND(-2)

SM_XVIRTUALSCREEN, SM_YVIRTUALSCREEN, SM_CXVIRTUALSCREEN, SM_CYVIRTUALSCREEN = 76, 77, 78, 79
SM_CXSMICON = 49

ULW_ALPHA = 0x2
AC_SRC_OVER = 0x0
AC_SRC_ALPHA = 0x1
PM_REMOVE = 0x1
QS_ALLINPUT = 0x04FF
MWMO_INPUTAVAILABLE = 0x0004
INFINITE = 0xFFFFFFFF

MF_STRING = 0x0000
MF_CHECKED = 0x0008
MF_POPUP = 0x0010
MF_SEPARATOR = 0x0800
TPM_RIGHTBUTTON = 0x0002
TPM_BOTTOMALIGN = 0x0020
TPM_RETURNCMD = 0x0100

NIM_ADD, NIM_MODIFY, NIM_DELETE = 0, 1, 2
NIF_MESSAGE, NIF_ICON, NIF_TIP, NIF_INFO = 0x1, 0x2, 0x4, 0x10
NIIF_USER, NIIF_LARGE_ICON = 0x4, 0x20

DWMWA_EXTENDED_FRAME_BOUNDS = 9
DWMWA_CLOAKED = 14
WDA_NONE = 0x0
WDA_EXCLUDEFROMCAPTURE = 0x11
MB_ICONINFORMATION = 0x40
ERROR_ALREADY_EXISTS = 183

HKEY_CURRENT_USER = wt.HKEY(0x80000001)
KEY_QUERY_VALUE = 0x0001
KEY_SET_VALUE = 0x0002
REG_SZ = 1

QUNS_BUSY, QUNS_RUNNING_D3D_FULL_SCREEN, QUNS_PRESENTATION_MODE = 2, 3, 4


class WNDCLASSEXW(ctypes.Structure):
    _fields_ = [
        ("cbSize", wt.UINT), ("style", wt.UINT), ("lpfnWndProc", WNDPROC), ("cbClsExtra", ctypes.c_int),
        ("cbWndExtra", ctypes.c_int), ("hInstance", wt.HINSTANCE), ("hIcon", wt.HICON), ("hCursor", HCURSOR),
        ("hbrBackground", wt.HBRUSH), ("lpszMenuName", wt.LPCWSTR), ("lpszClassName", wt.LPCWSTR),
        ("hIconSm", wt.HICON),
    ]


class BITMAPINFOHEADER(ctypes.Structure):
    _fields_ = [
        ("biSize", wt.DWORD), ("biWidth", wt.LONG), ("biHeight", wt.LONG), ("biPlanes", wt.WORD),
        ("biBitCount", wt.WORD), ("biCompression", wt.DWORD), ("biSizeImage", wt.DWORD),
        ("biXPelsPerMeter", wt.LONG), ("biYPelsPerMeter", wt.LONG), ("biClrUsed", wt.DWORD),
        ("biClrImportant", wt.DWORD),
    ]


class BITMAPINFO(ctypes.Structure):
    _fields_ = [("bmiHeader", BITMAPINFOHEADER), ("bmiColors", wt.DWORD * 1)]


class BLENDFUNCTION(ctypes.Structure):
    _fields_ = [("BlendOp", wt.BYTE), ("BlendFlags", wt.BYTE), ("SourceConstantAlpha", wt.BYTE),
                ("AlphaFormat", wt.BYTE)]


class MONITORINFO(ctypes.Structure):
    _fields_ = [("cbSize", wt.DWORD), ("rcMonitor", wt.RECT), ("rcWork", wt.RECT), ("dwFlags", wt.DWORD)]


class ICONINFO(ctypes.Structure):
    _fields_ = [("fIcon", wt.BOOL), ("xHotspot", wt.DWORD), ("yHotspot", wt.DWORD),
                ("hbmMask", wt.HBITMAP), ("hbmColor", wt.HBITMAP)]


class GUID(ctypes.Structure):
    _fields_ = [("Data1", wt.DWORD), ("Data2", wt.WORD), ("Data3", wt.WORD), ("Data4", wt.BYTE * 8)]


class NOTIFYICONDATAW(ctypes.Structure):
    _fields_ = [
        ("cbSize", wt.DWORD), ("hWnd", wt.HWND), ("uID", wt.UINT), ("uFlags", wt.UINT),
        ("uCallbackMessage", wt.UINT), ("hIcon", wt.HICON), ("szTip", wt.WCHAR * 128), ("dwState", wt.DWORD),
        ("dwStateMask", wt.DWORD), ("szInfo", wt.WCHAR * 256), ("uVersion", wt.UINT),
        ("szInfoTitle", wt.WCHAR * 64), ("dwInfoFlags", wt.DWORD), ("guidItem", GUID),
        ("hBalloonIcon", wt.HICON),
    ]


def _fn(dll, name, restype, *argtypes, optional=False):
    try:
        f = getattr(dll, name)
    except AttributeError:
        if optional:
            return None
        raise
    f.restype = restype
    f.argtypes = argtypes
    return f


P = ctypes.POINTER

RegisterClassExW = _fn(user32, "RegisterClassExW", wt.ATOM, P(WNDCLASSEXW))
CreateWindowExW = _fn(user32, "CreateWindowExW", wt.HWND, wt.DWORD, wt.LPCWSTR, wt.LPCWSTR, wt.DWORD,
                      ctypes.c_int, ctypes.c_int, ctypes.c_int, ctypes.c_int, wt.HWND, wt.HMENU,
                      wt.HINSTANCE, wt.LPVOID)
DefWindowProcW = _fn(user32, "DefWindowProcW", LRESULT, wt.HWND, wt.UINT, wt.WPARAM, wt.LPARAM)
PeekMessageW = _fn(user32, "PeekMessageW", wt.BOOL, P(wt.MSG), wt.HWND, wt.UINT, wt.UINT, wt.UINT)
TranslateMessage = _fn(user32, "TranslateMessage", wt.BOOL, P(wt.MSG))
DispatchMessageW = _fn(user32, "DispatchMessageW", LRESULT, P(wt.MSG))
PostQuitMessage = _fn(user32, "PostQuitMessage", None, ctypes.c_int)
PostMessageW = _fn(user32, "PostMessageW", wt.BOOL, wt.HWND, wt.UINT, wt.WPARAM, wt.LPARAM)
MsgWaitForMultipleObjectsEx = _fn(user32, "MsgWaitForMultipleObjectsEx", wt.DWORD, wt.DWORD, P(wt.HANDLE),
                                  wt.DWORD, wt.DWORD, wt.DWORD)
GetCursorPos = _fn(user32, "GetCursorPos", wt.BOOL, P(wt.POINT))
EnumWindows = _fn(user32, "EnumWindows", wt.BOOL, WNDENUMPROC, wt.LPARAM)
GetForegroundWindow = _fn(user32, "GetForegroundWindow", wt.HWND)
GetWindow = _fn(user32, "GetWindow", wt.HWND, wt.HWND, wt.UINT)
GW_HWNDPREV = 3
IsWindowVisible = _fn(user32, "IsWindowVisible", wt.BOOL, wt.HWND)
IsIconic = _fn(user32, "IsIconic", wt.BOOL, wt.HWND)
IsZoomed = _fn(user32, "IsZoomed", wt.BOOL, wt.HWND)
# 32-bit Windows has only GetWindowLongW.
GetWindowLongPtrW = (_fn(user32, "GetWindowLongPtrW", LONG_PTR, wt.HWND, ctypes.c_int, optional=True)
                     or _fn(user32, "GetWindowLongW", wt.LONG, wt.HWND, ctypes.c_int))
GetClassNameW = _fn(user32, "GetClassNameW", ctypes.c_int, wt.HWND, wt.LPWSTR, ctypes.c_int)
GetWindowRect = _fn(user32, "GetWindowRect", wt.BOOL, wt.HWND, P(wt.RECT))
EnumDisplayMonitors = _fn(user32, "EnumDisplayMonitors", wt.BOOL, wt.HDC, P(wt.RECT), MONITORENUMPROC, wt.LPARAM)
GetMonitorInfoW = _fn(user32, "GetMonitorInfoW", wt.BOOL, wt.HMONITOR, P(MONITORINFO))
GetSystemMetrics = _fn(user32, "GetSystemMetrics", ctypes.c_int, ctypes.c_int)
UpdateLayeredWindow = _fn(user32, "UpdateLayeredWindow", wt.BOOL, wt.HWND, wt.HDC, P(wt.POINT), P(wt.SIZE),
                          wt.HDC, P(wt.POINT), wt.DWORD, P(BLENDFUNCTION), wt.DWORD)
SetWindowPos = _fn(user32, "SetWindowPos", wt.BOOL, wt.HWND, wt.HWND, ctypes.c_int, ctypes.c_int, ctypes.c_int,
                   ctypes.c_int, wt.UINT)
ShowWindow = _fn(user32, "ShowWindow", wt.BOOL, wt.HWND, ctypes.c_int)
GetDC = _fn(user32, "GetDC", wt.HDC, wt.HWND)
ReleaseDC = _fn(user32, "ReleaseDC", ctypes.c_int, wt.HWND, wt.HDC)
CreatePopupMenu = _fn(user32, "CreatePopupMenu", wt.HMENU)
AppendMenuW = _fn(user32, "AppendMenuW", wt.BOOL, wt.HMENU, wt.UINT, UINT_PTR, wt.LPCWSTR)
TrackPopupMenu = _fn(user32, "TrackPopupMenu", wt.BOOL, wt.HMENU, wt.UINT, ctypes.c_int, ctypes.c_int,
                     ctypes.c_int, wt.HWND, wt.LPVOID)
DestroyMenu = _fn(user32, "DestroyMenu", wt.BOOL, wt.HMENU)
SetForegroundWindow = _fn(user32, "SetForegroundWindow", wt.BOOL, wt.HWND)
RegisterWindowMessageW = _fn(user32, "RegisterWindowMessageW", wt.UINT, wt.LPCWSTR)
CreateIconIndirect = _fn(user32, "CreateIconIndirect", wt.HICON, P(ICONINFO))
MessageBoxW = _fn(user32, "MessageBoxW", ctypes.c_int, wt.HWND, wt.LPCWSTR, wt.LPCWSTR, wt.UINT)
LoadCursorW = _fn(user32, "LoadCursorW", HCURSOR, wt.HINSTANCE, wt.LPVOID)
SetTimer = _fn(user32, "SetTimer", UINT_PTR, wt.HWND, UINT_PTR, wt.UINT, wt.LPVOID)
KillTimer = _fn(user32, "KillTimer", wt.BOOL, wt.HWND, UINT_PTR)
SetWindowDisplayAffinity = _fn(user32, "SetWindowDisplayAffinity", wt.BOOL, wt.HWND, wt.DWORD, optional=True)
SetProcessDpiAwarenessContext = _fn(user32, "SetProcessDpiAwarenessContext", wt.BOOL, wt.HANDLE, optional=True)
SetProcessDPIAware = _fn(user32, "SetProcessDPIAware", wt.BOOL, optional=True)
GetDpiForSystem = _fn(user32, "GetDpiForSystem", wt.UINT, optional=True)

CreateCompatibleDC = _fn(gdi32, "CreateCompatibleDC", wt.HDC, wt.HDC)
CreateDIBSection = _fn(gdi32, "CreateDIBSection", wt.HBITMAP, wt.HDC, P(BITMAPINFO), wt.UINT, P(wt.LPVOID),
                       wt.HANDLE, wt.DWORD)
CreateBitmap = _fn(gdi32, "CreateBitmap", wt.HBITMAP, ctypes.c_int, ctypes.c_int, wt.UINT, wt.UINT, wt.LPVOID)
SelectObject = _fn(gdi32, "SelectObject", wt.HGDIOBJ, wt.HDC, wt.HGDIOBJ)
DeleteObject = _fn(gdi32, "DeleteObject", wt.BOOL, wt.HGDIOBJ)
DeleteDC = _fn(gdi32, "DeleteDC", wt.BOOL, wt.HDC)

GetModuleHandleW = _fn(kernel32, "GetModuleHandleW", wt.HMODULE, wt.LPCWSTR)
CreateMutexW = _fn(kernel32, "CreateMutexW", wt.HANDLE, wt.LPVOID, wt.BOOL, wt.LPCWSTR)
CreateWaitableTimerExW = _fn(kernel32, "CreateWaitableTimerExW", wt.HANDLE, wt.LPVOID, wt.LPCWSTR, wt.DWORD,
                             wt.DWORD, optional=True)
SetWaitableTimer = _fn(kernel32, "SetWaitableTimer", wt.BOOL, wt.HANDLE, P(wt.LARGE_INTEGER), wt.LONG, wt.LPVOID,
                       wt.LPVOID, wt.BOOL)
GetUserDefaultUILanguage = _fn(kernel32, "GetUserDefaultUILanguage", wt.WORD)

Shell_NotifyIconW = _fn(shell32, "Shell_NotifyIconW", wt.BOOL, wt.DWORD, P(NOTIFYICONDATAW))
SHQueryUserNotificationState = _fn(shell32, "SHQueryUserNotificationState", ctypes.c_long, P(ctypes.c_int),
                                   optional=True)

DwmGetWindowAttribute = _fn(dwmapi, "DwmGetWindowAttribute", ctypes.c_long, wt.HWND, wt.DWORD, wt.LPVOID, wt.DWORD)

RegOpenKeyExW = _fn(advapi32, "RegOpenKeyExW", wt.LONG, wt.HKEY, wt.LPCWSTR, wt.DWORD, wt.DWORD, P(wt.HKEY))
RegSetValueExW = _fn(advapi32, "RegSetValueExW", wt.LONG, wt.HKEY, wt.LPCWSTR, wt.DWORD, wt.DWORD, wt.LPVOID,
                     wt.DWORD)
RegDeleteValueW = _fn(advapi32, "RegDeleteValueW", wt.LONG, wt.HKEY, wt.LPCWSTR)
RegQueryValueExW = _fn(advapi32, "RegQueryValueExW", wt.LONG, wt.HKEY, wt.LPCWSTR, wt.LPVOID, wt.LPVOID, wt.LPVOID,
                       wt.LPVOID)
RegCloseKey = _fn(advapi32, "RegCloseKey", wt.LONG, wt.HKEY)

WTSRegisterSessionNotification = _fn(wtsapi32, "WTSRegisterSessionNotification", wt.BOOL, wt.HWND, wt.DWORD)


# ---- helpers ---------------------------------------------------------------------------------------------------


def set_dpi_aware() -> None:
    """See physical pixels on every monitor."""
    if SetProcessDpiAwarenessContext and SetProcessDpiAwarenessContext(wt.HANDLE(-4)):  # PER_MONITOR_AWARE_V2
        return
    if SetProcessDPIAware:
        SetProcessDPIAware()


def system_scale() -> float:
    dpi = GetDpiForSystem() if GetDpiForSystem else 96
    return (dpi or 96) / 96


def single_instance(name: str) -> bool:
    """Hold a named mutex; False if another copy already does."""
    handle = CreateMutexW(None, False, name)
    if handle and ctypes.get_last_error() == ERROR_ALREADY_EXISTS:
        return False
    single_instance.handle = handle  # keep it for the life of the process
    return True


def virtual_screen() -> tuple[int, int, int, int]:
    return (GetSystemMetrics(SM_XVIRTUALSCREEN), GetSystemMetrics(SM_YVIRTUALSCREEN),
            GetSystemMetrics(SM_CXVIRTUALSCREEN), GetSystemMetrics(SM_CYVIRTUALSCREEN))


def window_frame(hwnd) -> wt.RECT | None:
    """The visible frame of a window, without the invisible resize border."""
    r = wt.RECT()
    if DwmGetWindowAttribute(hwnd, DWMWA_EXTENDED_FRAME_BOUNDS, ctypes.byref(r), ctypes.sizeof(r)) == 0:
        return r
    return r if GetWindowRect(hwnd, ctypes.byref(r)) else None


def is_cloaked(hwnd) -> bool:
    """DWM hides the window, as it does for windows on other virtual desktops."""
    v = wt.DWORD()
    return DwmGetWindowAttribute(hwnd, DWMWA_CLOAKED, ctypes.byref(v), 4) == 0 and v.value != 0


def class_name(hwnd) -> str:
    buf = ctypes.create_unicode_buffer(128)
    GetClassNameW(hwnd, buf, 128)
    return buf.value


def create_dib(width: int, height: int):
    """A top-down 32-bit bitmap and the address of its pixels."""
    bi = BITMAPINFO()
    bi.bmiHeader.biSize = ctypes.sizeof(BITMAPINFOHEADER)
    bi.bmiHeader.biWidth = width
    bi.bmiHeader.biHeight = -height
    bi.bmiHeader.biPlanes = 1
    bi.bmiHeader.biBitCount = 32
    bits = wt.LPVOID()
    bmp = CreateDIBSection(None, ctypes.byref(bi), 0, ctypes.byref(bits), None, 0)
    if not bmp or not bits.value:
        return None, None
    return bmp, bits.value


def create_icon(pixels, size: int):
    """An icon from (size, size) premultiplied 0xAARRGGBB uint32 pixels."""
    bmp, bits = create_dib(size, size)
    if not bmp:
        return None
    ctypes.memmove(bits, pixels.tobytes(), size * size * 4)
    mask = CreateBitmap(size, size, 1, 1, None)
    info = ICONINFO(True, 0, 0, mask, bmp)
    icon = CreateIconIndirect(ctypes.byref(info))
    DeleteObject(bmp)
    DeleteObject(mask)
    return icon


def update_layered(hwnd, src_dc, x: int, y: int, w: int, h: int) -> bool:
    blend = BLENDFUNCTION(AC_SRC_OVER, 0, 255, AC_SRC_ALPHA)
    return bool(UpdateLayeredWindow(hwnd, None, ctypes.byref(wt.POINT(x, y)), ctypes.byref(wt.SIZE(w, h)), src_dc,
                                    ctypes.byref(wt.POINT(0, 0)), 0, ctypes.byref(blend), ULW_ALPHA))


def exclude_from_capture(hwnd, exclude: bool) -> None:
    """Keep the window out of screenshots and screen sharing (Windows 10 2004 and later)."""
    if SetWindowDisplayAffinity:
        SetWindowDisplayAffinity(hwnd, WDA_EXCLUDEFROMCAPTURE if exclude else WDA_NONE)


def shell_busy() -> bool:
    """A full-screen Direct3D game or a presentation is running.

    QUNS_BUSY is left out on purpose: the shell reports it for any window that covers a whole monitor, and the
    snow overlay itself is one, so the snow would keep pausing itself and blink."""
    if not SHQueryUserNotificationState:
        return False
    state = ctypes.c_int()
    if SHQueryUserNotificationState(ctypes.byref(state)) != 0:
        return False
    return state.value in (QUNS_RUNNING_D3D_FULL_SCREEN, QUNS_PRESENTATION_MODE)


def new_frame_timer():
    """A high-resolution waitable timer where Windows has one (10 1803 and later), else an ordinary one."""
    if not CreateWaitableTimerExW:
        return None
    access = 0x1F0003  # TIMER_ALL_ACCESS
    return CreateWaitableTimerExW(None, None, 0x2, access) or CreateWaitableTimerExW(None, None, 0, access) or None


def wait(timer, seconds: float) -> None:
    """Sleep until a message arrives or the time passes."""
    if timer and seconds < 0.2:
        due = wt.LARGE_INTEGER(-max(int(seconds * 1e7), 1))
        SetWaitableTimer(timer, ctypes.byref(due), 0, None, None, False)
        handles = (wt.HANDLE * 1)(timer)
        MsgWaitForMultipleObjectsEx(1, handles, INFINITE, QS_ALLINPUT, MWMO_INPUTAVAILABLE)
    else:
        MsgWaitForMultipleObjectsEx(0, None, max(int(seconds * 1000), 1), QS_ALLINPUT, MWMO_INPUTAVAILABLE)


RUN_KEY = r"Software\Microsoft\Windows\CurrentVersion\Run"


def _open_run(access: int):
    key = wt.HKEY()
    if RegOpenKeyExW(HKEY_CURRENT_USER, RUN_KEY, 0, access, ctypes.byref(key)) != 0:
        return None
    return key


def run_at_login(name: str) -> bool:
    key = _open_run(KEY_QUERY_VALUE)
    if not key:
        return False
    try:
        return RegQueryValueExW(key, name, None, None, None, None) == 0
    finally:
        RegCloseKey(key)


def set_run_at_login(name: str, command: str, on: bool) -> bool:
    key = _open_run(KEY_SET_VALUE)
    if not key:
        return False
    try:
        if not on:
            return RegDeleteValueW(key, name) == 0
        data = ctypes.create_unicode_buffer(command)
        return RegSetValueExW(key, name, 0, REG_SZ, data, ctypes.sizeof(data)) == 0
    finally:
        RegCloseKey(key)
