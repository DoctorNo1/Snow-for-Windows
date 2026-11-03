//go:build windows

// Package app is the Windows app: a click-through overlay over every monitor that the snow is drawn into, a
// watcher that tells the simulation where the windows are, and the tray icon with its menu.
package app

import (
	"runtime"
	"syscall"
	"time"

	"github.com/doctorno1/snow-for-windows/internal/icon"
	"github.com/doctorno1/snow-for-windows/internal/snow"
	"github.com/doctorno1/snow-for-windows/internal/win"
)

const (
	runName   = "SnowForWindows"
	trayMsg   = win.WM_APP + 1
	menuTimer = 1
)

// Menu command ids.
const (
	cmdIntensity = 100 + iota // followed by one id per intensity level
	_
	_
	_
	_
	_
	cmdAccumulate
	cmdMelt
	cmdOnWindows
	cmdInScreenshot
	cmdSmooth
	cmdStartAtLogin
	cmdClear
	cmdAbout
	cmdQuit
)

type monitor struct{ full, work win.RECT }

type tracked struct {
	hwnd    uintptr
	rect    win.RECT
	surface bool
}

type App struct {
	t     *Text
	prefs Prefs

	inst    uintptr
	overlay uintptr
	tray    uintptr
	icon    uintptr

	memDC, bmp, oldBmp uintptr
	vx, vy, vw, vh     int32
	scale              float32

	world  *snow.World
	canvas *snow.Canvas

	monitors []monitor
	wins     []tracked

	shown          bool
	locked         bool
	busy           bool
	layoutDirty    bool
	menuOpen       bool
	taskbarCreated uint32

	last      time.Time
	nextTrack time.Time
	nextBusy  time.Time
	nextTop   time.Time
	nextMon   time.Time
}

var a *App

// Run starts the app and returns when the user quits.
func Run() {
	runtime.LockOSThread()
	win.SetDPIAware()
	if !win.SingleInstance(`Local\SnowForWindows-7c1d2f`) {
		return
	}
	a = &App{t: TextFor(win.UILanguage()), prefs: LoadPrefs(), inst: win.GetModuleHandle()}
	a.world = snow.NewWorld(1, 1, 1, uint64(time.Now().UnixNano()))
	if !a.createWindows() {
		win.MessageBox(0, "Could not create the snow window.", a.t.AppName)
		return
	}
	a.rebuild()
	a.apply(false)
	a.addTray()
	if !a.prefs.Welcomed {
		a.balloon(a.t.Welcome)
		a.prefs.Welcomed = true
		a.prefs.Save()
	}
	a.loop()
	a.removeTray()
	win.ShowWindow(a.overlay, win.SW_HIDE)
}

func (a *App) createWindows() bool {
	overlayClass := win.WNDCLASSEX{
		WndProc:   syscall.NewCallback(overlayProc),
		Instance:  a.inst,
		Cursor:    win.LoadArrowCursor(),
		ClassName: win.Str("SnowForWindowsOverlay"),
	}
	trayClass := win.WNDCLASSEX{
		WndProc:   syscall.NewCallback(trayProc),
		Instance:  a.inst,
		ClassName: win.Str("SnowForWindowsTray"),
	}
	if win.RegisterClassEx(&overlayClass) == 0 || win.RegisterClassEx(&trayClass) == 0 {
		return false
	}
	ex := uint32(win.WS_EX_LAYERED | win.WS_EX_TRANSPARENT | win.WS_EX_TOOLWINDOW | win.WS_EX_NOACTIVATE | win.WS_EX_TOPMOST)
	a.overlay = win.CreateWindowEx(ex, "SnowForWindowsOverlay", "Snow", win.WS_POPUP, 0, 0, 1, 1, 0, a.inst)
	a.tray = win.CreateWindowEx(win.WS_EX_TOOLWINDOW, "SnowForWindowsTray", "Snow for Windows", win.WS_POPUP, 0, 0, 0, 0, 0, a.inst)
	if a.overlay == 0 || a.tray == 0 {
		return false
	}
	a.taskbarCreated = win.RegisterWindowMessage("TaskbarCreated")
	win.RegisterSessionNotification(a.overlay)
	size := int(win.GetSystemMetrics(win.SM_CXSMICON))
	if size < 16 {
		size = 16
	}
	a.icon = win.CreateIcon(icon.BGRA(size), size)
	return true
}

func overlayProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch uint32(msg) {
	case win.WM_NCHITTEST:
		return win.HTTRANSPARENT
	case win.WM_DISPLAYCHANGE, win.WM_DPICHANGED, win.WM_SETTINGCHANGE:
		if a != nil {
			a.layoutDirty = true
		}
		return 0
	case win.WM_WTSSESSION_CHANGE:
		if a != nil {
			switch wp {
			case win.WTS_SESSION_LOCK:
				a.locked = true
			case win.WTS_SESSION_UNLOCK:
				a.locked = false
			}
		}
		return 0
	}
	return win.DefWindowProc(hwnd, uint32(msg), wp, lp)
}

func trayProc(hwnd, msg, wp, lp uintptr) uintptr {
	m := uint32(msg)
	switch {
	case m == trayMsg:
		switch uint32(lp & 0xffff) {
		case win.WM_LBUTTONUP, win.WM_RBUTTONUP:
			a.showMenu()
		}
		return 0
	case m == win.WM_TIMER && wp == menuTimer:
		// The menu runs its own message loop; keep the snow moving while it is open.
		a.frame()
		return 0
	case a != nil && m == a.taskbarCreated && m != 0:
		// Explorer restarted: the tray is new and empty.
		a.addTray()
		return 0
	case m == win.WM_DISPLAYCHANGE || m == win.WM_SETTINGCHANGE:
		a.layoutDirty = true
	}
	return win.DefWindowProc(hwnd, m, wp, lp)
}

// rebuild sizes the overlay to the virtual screen and finds the ground along each monitor.
func (a *App) rebuild() {
	a.layoutDirty = false
	x := win.GetSystemMetrics(win.SM_XVIRTUALSCREEN)
	y := win.GetSystemMetrics(win.SM_YVIRTUALSCREEN)
	w := win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN)
	h := win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN)
	if w <= 0 || h <= 0 {
		return
	}
	scale := win.SystemScale()
	if w != a.vw || h != a.vh || scale != a.scale || a.memDC == 0 {
		a.freeBitmap()
		screen := win.GetDC(0)
		a.memDC = win.CreateCompatibleDC(screen)
		win.ReleaseDC(0, screen)
		var pix []uint32
		a.bmp, pix = win.CreateDIB(int(w), int(h))
		if a.bmp == 0 {
			return
		}
		a.oldBmp = win.SelectObject(a.memDC, a.bmp)
		a.canvas = snow.NewCanvas(pix, int(w), int(h))
		a.world.Resize(int(w), int(h), scale)
		a.vw, a.vh, a.scale = w, h, scale
	}
	a.vx, a.vy = x, y
	a.monitors = a.monitors[:0]
	win.Monitors(monitorCallback)
	a.canvas.Invalidate()
	a.nextTrack = time.Time{}
}

var monitorCallback = syscall.NewCallback(func(hmon, hdc, rc, lp uintptr) uintptr {
	if mi, ok := win.GetMonitorInfo(hmon); ok {
		a.monitors = append(a.monitors, monitor{mi.Monitor, mi.Work})
	}
	return 1
})

func (a *App) freeBitmap() {
	if a.memDC != 0 {
		win.SelectObject(a.memDC, a.oldBmp)
		win.DeleteDC(a.memDC)
		a.memDC = 0
	}
	if a.bmp != 0 {
		win.DeleteObject(a.bmp)
		a.bmp = 0
	}
}

// Shell windows that never hold snow: the desktop, the taskbar, shadows and tooltips.
var ignored = map[string]bool{
	"Progman": true, "WorkerW": true, "Shell_TrayWnd": true, "Shell_SecondaryTrayWnd": true,
	"tooltips_class32": true, "SysShadow": true, "EdgeUiInputTopWndClass": true, "EdgeUiInputWndClass": true,
	"ApplicationManager_ImmersiveShellWindow": true, "Internet Explorer_Hidden": true,
	"SnowForWindowsOverlay": true, "SnowForWindowsTray": true,
}

// Switchers, menus and shell flyouts hide the snow behind them but don't collect any.
var coverOnly = map[string]bool{
	"XamlExplorerHostIslandWindow": true, "MultitaskingViewFrame": true, "TaskListThumbnailWnd": true,
	"ForegroundStaging": true, "TopLevelWindowForOverflowXamlIsland": true, "NotifyIconOverflowWindow": true,
	"Xaml_WindowedPopupClass": true, "Windows.UI.Core.CoreWindow": true, "#32768": true,
}

var enumCallback = syscall.NewCallback(func(hwnd, lp uintptr) uintptr {
	if hwnd == a.overlay || hwnd == a.tray || !win.IsWindowVisible(hwnd) || win.IsIconic(hwnd) || win.IsCloaked(hwnd) {
		return 1
	}
	class := win.GetClassName(hwnd)
	if ignored[class] {
		return 1
	}
	ex := win.GetWindowLong(hwnd, win.GWL_EXSTYLE)
	if ex&win.WS_EX_LAYERED != 0 && ex&win.WS_EX_TRANSPARENT != 0 {
		return 1 // another click-through overlay
	}
	r, ok := win.WindowFrame(hwnd)
	if !ok || r.Right-r.Left < 2 || r.Bottom-r.Top < 2 {
		return 1
	}
	sc := a.scale
	surface := !coverOnly[class] && ex&win.WS_EX_TOOLWINDOW == 0 && !win.IsZoomed(hwnd) &&
		float32(r.Right-r.Left) >= 120*sc && float32(r.Bottom-r.Top) >= 50*sc
	if surface {
		// No snow on a window whose top is at the top of its monitor, as a maximized or snapped one's is.
		cx := (r.Left + r.Right) / 2
		for _, m := range a.monitors {
			if cx >= m.full.Left && cx < m.full.Right && r.Top >= m.full.Top && r.Top < m.full.Bottom {
				if r.Top <= m.work.Top+4 {
					surface = false
				}
				break
			}
		}
	}
	a.wins = append(a.wins, tracked{hwnd, r, surface})
	return 1
})

// track tells the simulation where the windows and the ground are. Windows come in z order, front first.
func (a *App) track() {
	a.wins = a.wins[:0]
	win.EnumWindows(enumCallback)
	occ := make([]snow.Rect, 0, len(a.wins))
	specs := make([]snow.SurfaceSpec, 0, len(a.wins)+len(a.monitors))
	inset := int32(3 * a.scale) // stay clear of the rounded corners
	for i, w := range a.wins {
		r := a.toBuffer(w.rect)
		occ = append(occ, r)
		if w.surface && a.prefs.OnWindows {
			specs = append(specs, snow.SurfaceSpec{
				ID: uint64(w.hwnd), L: r.L + int(inset), R: r.R - int(inset), Top: r.T, Z: i,
			})
		}
	}
	for i, m := range a.monitors {
		r := a.toBuffer(m.work)
		specs = append(specs, snow.SurfaceSpec{
			ID: 1<<63 | uint64(i), L: r.L, R: r.R, Top: r.B, Ground: true, Z: len(occ),
		})
	}
	a.world.SetLayout(specs, occ)
}

func (a *App) toBuffer(r win.RECT) snow.Rect {
	return snow.Rect{L: int(r.Left - a.vx), T: int(r.Top - a.vy), R: int(r.Right - a.vx), B: int(r.Bottom - a.vy)}
}

func (a *App) show() {
	if !a.shown {
		win.ShowWindow(a.overlay, win.SW_SHOWNOACTIVATE)
		win.SetWindowPos(a.overlay, win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOACTIVATE)
		a.shown = true
	}
}

func (a *App) hide() {
	if a.shown {
		win.ShowWindow(a.overlay, win.SW_HIDE)
		a.shown = false
		if a.canvas != nil {
			a.canvas.Invalidate()
		}
	}
}

func (a *App) frameInterval() time.Duration {
	if a.prefs.Smooth {
		return time.Second / 60
	}
	return time.Second / 30
}

// frame advances and draws one frame. It returns how long the loop may sleep before the next one.
func (a *App) frame() time.Duration {
	now := time.Now()
	dt := float32(now.Sub(a.last).Seconds())
	if a.last.IsZero() {
		dt = 0
	}
	a.last = now
	if a.layoutDirty || now.After(a.nextMon) {
		if a.layoutDirty {
			a.rebuild()
		} else {
			a.checkMonitors()
		}
		a.nextMon = now.Add(3 * time.Second)
	}
	if now.After(a.nextBusy) {
		a.busy = win.ShellBusy()
		a.nextBusy = now.Add(500 * time.Millisecond)
	}
	if a.locked || a.busy || a.canvas == nil {
		a.hide()
		a.last = time.Time{}
		return 500 * time.Millisecond
	}
	if now.After(a.nextTrack) {
		a.track()
		a.nextTrack = now.Add(60 * time.Millisecond)
	}
	if p, ok := win.GetCursorPos(); ok {
		a.world.SetCursor(float32(p.X-a.vx), float32(p.Y-a.vy), true, dt)
	} else {
		a.world.SetCursor(0, 0, false, dt)
	}
	a.world.Update(dt)
	if a.world.Idle() {
		a.hide()
		a.last = time.Time{}
		return time.Second
	}
	a.world.Draw(a.canvas)
	win.UpdateLayeredWindow(a.overlay, a.memDC, win.POINT{X: a.vx, Y: a.vy}, win.SIZE{CX: a.vw, CY: a.vh})
	a.show()
	if now.After(a.nextTop) {
		// Other topmost windows come and go; stay in front of them.
		win.SetWindowPos(a.overlay, win.HWND_TOPMOST, 0, 0, 0, 0, win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOACTIVATE)
		a.nextTop = now.Add(2 * time.Second)
	}
	wait := a.frameInterval() - time.Since(now)
	if wait < time.Millisecond {
		wait = time.Millisecond
	}
	return wait
}

// checkMonitors rebuilds the layout if the screens changed without a message saying so.
func (a *App) checkMonitors() {
	x := win.GetSystemMetrics(win.SM_XVIRTUALSCREEN)
	y := win.GetSystemMetrics(win.SM_YVIRTUALSCREEN)
	w := win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN)
	h := win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN)
	old := append([]monitor(nil), a.monitors...)
	a.monitors = a.monitors[:0]
	win.Monitors(monitorCallback)
	changed := x != a.vx || y != a.vy || w != a.vw || h != a.vh || len(old) != len(a.monitors) || win.SystemScale() != a.scale
	for i := 0; !changed && i < len(old); i++ {
		changed = old[i] != a.monitors[i]
	}
	if changed {
		a.rebuild()
	}
}

func (a *App) loop() {
	timer := win.NewFrameTimer()
	var msg win.MSG
	for {
		for win.PeekMessage(&msg) {
			if msg.Message == win.WM_QUIT {
				return
			}
			win.TranslateMessage(&msg)
			win.DispatchMessage(&msg)
		}
		wait := a.frame()
		if timer != 0 && wait < 200*time.Millisecond {
			win.ArmTimer(timer, int64(wait/100))
			win.MsgWait(timer, win.INFINITE)
		} else {
			win.MsgWait(0, uint32(wait/time.Millisecond))
		}
	}
}

// apply hands the settings to the simulation and the overlay.
func (a *App) apply(save bool) {
	level, variable := Level(a.prefs.Intensity)
	a.world.S = snow.Settings{
		Intensity:  level,
		Variable:   variable,
		Accumulate: a.prefs.Accumulate,
		Melt:       a.prefs.Melt,
	}
	win.SetExcludeFromCapture(a.overlay, !a.prefs.InScreenshot)
	a.nextTrack = time.Time{}
	if save {
		a.prefs.Save()
		a.updateTip()
	}
}

func (a *App) trayData() win.NOTIFYICONDATA {
	d := win.NOTIFYICONDATA{Wnd: a.tray, ID: 1, Flags: win.NIF_MESSAGE | win.NIF_ICON | win.NIF_TIP,
		CallbackMessage: trayMsg, Icon: a.icon}
	tip := a.t.Tip
	if a.prefs.Intensity == IntensityOff {
		tip = a.t.PausedTip
	}
	copyUTF16(d.Tip[:], tip)
	return d
}

func (a *App) addTray() {
	d := a.trayData()
	win.NotifyIcon(win.NIM_ADD, &d)
}

func (a *App) updateTip() {
	d := a.trayData()
	win.NotifyIcon(win.NIM_MODIFY, &d)
}

func (a *App) removeTray() {
	d := win.NOTIFYICONDATA{Wnd: a.tray, ID: 1}
	win.NotifyIcon(win.NIM_DELETE, &d)
}

// balloon shows a notification from the tray icon.
func (a *App) balloon(text string) {
	const nifInfo, niifUser, niifLargeIcon = 0x10, 0x4, 0x20
	d := a.trayData()
	d.Flags |= nifInfo
	d.InfoFlags = niifUser | niifLargeIcon
	d.BalloonIcon = a.icon
	copyUTF16(d.Info[:], text)
	copyUTF16(d.InfoTitle[:], a.t.AppName)
	win.NotifyIcon(win.NIM_MODIFY, &d)
}

func copyUTF16(dst []uint16, s string) {
	u, _ := syscall.UTF16FromString(s)
	if len(u) > len(dst) {
		u = u[:len(dst)]
		u[len(u)-1] = 0
	}
	copy(dst, u)
}

func (a *App) showMenu() {
	if a.menuOpen {
		return
	}
	a.menuOpen = true
	defer func() { a.menuOpen = false }()
	t := a.t
	check := func(on bool) uint32 {
		if on {
			return win.MF_STRING | win.MF_CHECKED
		}
		return win.MF_STRING
	}
	menu := win.CreatePopupMenu()
	sub := win.CreatePopupMenu()
	for i, name := range []string{t.Off, t.Light, t.Steady, t.Heavy, t.Blizzard, t.Variable} {
		win.AppendMenu(sub, check(a.prefs.Intensity == i), uintptr(cmdIntensity+i), name)
	}
	win.AppendMenu(menu, win.MF_POPUP, sub, t.Intensity)
	win.AppendMenu(menu, win.MF_SEPARATOR, 0, "")
	win.AppendMenu(menu, check(a.prefs.Accumulate), cmdAccumulate, t.Accumulate)
	win.AppendMenu(menu, check(a.prefs.Melt), cmdMelt, t.Melt)
	win.AppendMenu(menu, check(a.prefs.OnWindows), cmdOnWindows, t.OnWindows)
	win.AppendMenu(menu, win.MF_STRING, cmdClear, t.Clear)
	win.AppendMenu(menu, win.MF_SEPARATOR, 0, "")
	win.AppendMenu(menu, check(a.prefs.InScreenshot), cmdInScreenshot, t.InScreenshot)
	win.AppendMenu(menu, check(a.prefs.Smooth), cmdSmooth, t.Smooth)
	win.AppendMenu(menu, check(win.RunAtLogin(runName)), cmdStartAtLogin, t.StartAtLogin)
	win.AppendMenu(menu, win.MF_SEPARATOR, 0, "")
	win.AppendMenu(menu, win.MF_STRING, cmdAbout, t.About)
	win.AppendMenu(menu, win.MF_STRING, cmdQuit, t.Quit)

	p, _ := win.GetCursorPos()
	win.SetForegroundWindow(a.tray)
	win.SetTimer(a.tray, menuTimer, uint32(a.frameInterval()/time.Millisecond))
	cmd := win.TrackPopupMenu(menu, win.TPM_RIGHTBUTTON|win.TPM_RETURNCMD|win.TPM_BOTTOMALIGN, p.X, p.Y, a.tray)
	win.KillTimer(a.tray, menuTimer)
	win.PostMessage(a.tray, win.WM_NULL, 0, 0)
	win.DestroyMenu(menu)
	a.command(int(cmd))
}

func (a *App) command(cmd int) {
	switch {
	case cmd >= cmdIntensity && cmd <= cmdIntensity+IntensityVariable:
		a.prefs.Intensity = cmd - cmdIntensity
	case cmd == cmdAccumulate:
		a.prefs.Accumulate = !a.prefs.Accumulate
	case cmd == cmdMelt:
		a.prefs.Melt = !a.prefs.Melt
	case cmd == cmdOnWindows:
		a.prefs.OnWindows = !a.prefs.OnWindows
	case cmd == cmdInScreenshot:
		a.prefs.InScreenshot = !a.prefs.InScreenshot
	case cmd == cmdSmooth:
		a.prefs.Smooth = !a.prefs.Smooth
	case cmd == cmdStartAtLogin:
		on := !win.RunAtLogin(runName)
		win.SetRunAtLogin(runName, `"`+win.ExePath()+`"`, on)
		return
	case cmd == cmdClear:
		a.world.Clear()
		return
	case cmd == cmdAbout:
		win.MessageBox(a.tray, a.t.AboutText, a.t.AppName)
		return
	case cmd == cmdQuit:
		win.PostQuitMessage(0)
		return
	default:
		return
	}
	a.apply(true)
}
