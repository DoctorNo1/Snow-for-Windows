//go:build windows

// Package win holds the few Win32 calls the app needs, made through syscall so the build needs no cgo and no
// third-party packages.
package win

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	wtsapi32 = syscall.NewLazyDLL("wtsapi32.dll")

	pRegisterClassExW              = user32.NewProc("RegisterClassExW")
	pCreateWindowExW               = user32.NewProc("CreateWindowExW")
	pDefWindowProcW                = user32.NewProc("DefWindowProcW")
	pDestroyWindow                 = user32.NewProc("DestroyWindow")
	pPeekMessageW                  = user32.NewProc("PeekMessageW")
	pTranslateMessage              = user32.NewProc("TranslateMessage")
	pDispatchMessageW              = user32.NewProc("DispatchMessageW")
	pPostQuitMessage               = user32.NewProc("PostQuitMessage")
	pPostMessageW                  = user32.NewProc("PostMessageW")
	pMsgWaitForMultipleObjectsEx   = user32.NewProc("MsgWaitForMultipleObjectsEx")
	pGetCursorPos                  = user32.NewProc("GetCursorPos")
	pEnumWindows                   = user32.NewProc("EnumWindows")
	pIsWindowVisible               = user32.NewProc("IsWindowVisible")
	pIsIconic                      = user32.NewProc("IsIconic")
	pIsZoomed                      = user32.NewProc("IsZoomed")
	pGetWindowLongPtrW             = user32.NewProc("GetWindowLongPtrW")
	pGetClassNameW                 = user32.NewProc("GetClassNameW")
	pGetWindowRect                 = user32.NewProc("GetWindowRect")
	pEnumDisplayMonitors           = user32.NewProc("EnumDisplayMonitors")
	pGetMonitorInfoW               = user32.NewProc("GetMonitorInfoW")
	pMonitorFromWindow             = user32.NewProc("MonitorFromWindow")
	pGetSystemMetrics              = user32.NewProc("GetSystemMetrics")
	pUpdateLayeredWindow           = user32.NewProc("UpdateLayeredWindow")
	pSetWindowPos                  = user32.NewProc("SetWindowPos")
	pShowWindow                    = user32.NewProc("ShowWindow")
	pGetDC                         = user32.NewProc("GetDC")
	pReleaseDC                     = user32.NewProc("ReleaseDC")
	pCreatePopupMenu               = user32.NewProc("CreatePopupMenu")
	pAppendMenuW                   = user32.NewProc("AppendMenuW")
	pTrackPopupMenu                = user32.NewProc("TrackPopupMenu")
	pDestroyMenu                   = user32.NewProc("DestroyMenu")
	pSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	pRegisterWindowMessageW        = user32.NewProc("RegisterWindowMessageW")
	pCreateIconIndirect            = user32.NewProc("CreateIconIndirect")
	pDestroyIcon                   = user32.NewProc("DestroyIcon")
	pSetWindowDisplayAffinity      = user32.NewProc("SetWindowDisplayAffinity")
	pSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	pSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	pGetDpiForSystem               = user32.NewProc("GetDpiForSystem")
	pMessageBoxW                   = user32.NewProc("MessageBoxW")
	pLoadCursorW                   = user32.NewProc("LoadCursorW")
	pSetTimer                      = user32.NewProc("SetTimer")
	pKillTimer                     = user32.NewProc("KillTimer")

	pCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	pCreateBitmap       = gdi32.NewProc("CreateBitmap")
	pSelectObject       = gdi32.NewProc("SelectObject")
	pDeleteObject       = gdi32.NewProc("DeleteObject")
	pDeleteDC           = gdi32.NewProc("DeleteDC")

	pGetModuleHandleW         = kernel32.NewProc("GetModuleHandleW")
	pCreateMutexW             = kernel32.NewProc("CreateMutexW")
	pCreateWaitableTimerExW   = kernel32.NewProc("CreateWaitableTimerExW")
	pSetWaitableTimer         = kernel32.NewProc("SetWaitableTimer")
	pGetUserDefaultUILanguage = kernel32.NewProc("GetUserDefaultUILanguage")
	pGetModuleFileNameW       = kernel32.NewProc("GetModuleFileNameW")

	pShellNotifyIconW             = shell32.NewProc("Shell_NotifyIconW")
	pSHQueryUserNotificationState = shell32.NewProc("SHQueryUserNotificationState")

	pDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute")

	pRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	pRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	pRegDeleteValueW  = advapi32.NewProc("RegDeleteValueW")
	pRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	pRegCloseKey      = advapi32.NewProc("RegCloseKey")

	pWTSRegisterSessionNotification = wtsapi32.NewProc("WTSRegisterSessionNotification")
)

// Window messages, styles and flags.
const (
	WM_DESTROY           = 0x0002
	WM_CLOSE             = 0x0010
	WM_QUIT              = 0x0012
	WM_DISPLAYCHANGE     = 0x007E
	WM_NCHITTEST         = 0x0084
	WM_COMMAND           = 0x0111
	WM_SETTINGCHANGE     = 0x001A
	WM_LBUTTONUP         = 0x0202
	WM_RBUTTONUP         = 0x0205
	WM_DPICHANGED        = 0x02E0
	WM_WTSSESSION_CHANGE = 0x02B1
	WM_NULL              = 0x0000
	WM_TIMER             = 0x0113
	WM_APP               = 0x8000

	WTS_SESSION_LOCK   = 0x7
	WTS_SESSION_UNLOCK = 0x8

	HTTRANSPARENT = ^uintptr(0) // -1

	WS_POPUP          = 0x80000000
	WS_EX_TOPMOST     = 0x00000008
	WS_EX_TRANSPARENT = 0x00000020
	WS_EX_TOOLWINDOW  = 0x00000080
	WS_EX_LAYERED     = 0x00080000
	WS_EX_NOACTIVATE  = 0x08000000
	WS_EX_APPWINDOW   = 0x00040000
	WS_CAPTION        = 0x00C00000

	GWL_STYLE   = ^uintptr(15) // -16
	GWL_EXSTYLE = ^uintptr(19) // -20

	SW_HIDE           = 0
	SW_SHOWNOACTIVATE = 4

	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOACTIVATE = 0x0010
	SWP_SHOWWINDOW = 0x0040

	HWND_TOPMOST = ^uintptr(0) // -1

	SM_XVIRTUALSCREEN  = 76
	SM_YVIRTUALSCREEN  = 77
	SM_CXVIRTUALSCREEN = 78
	SM_CYVIRTUALSCREEN = 79
	SM_CXSMICON        = 49

	ULW_ALPHA    = 0x2
	AC_SRC_OVER  = 0x0
	AC_SRC_ALPHA = 0x1

	PM_REMOVE = 0x1

	QS_ALLINPUT         = 0x04FF
	MWMO_INPUTAVAILABLE = 0x0004
	INFINITE            = 0xFFFFFFFF

	MF_STRING    = 0x0000
	MF_CHECKED   = 0x0008
	MF_POPUP     = 0x0010
	MF_SEPARATOR = 0x0800
	MF_GRAYED    = 0x0001

	TPM_RIGHTBUTTON = 0x0002
	TPM_RETURNCMD   = 0x0100
	TPM_BOTTOMALIGN = 0x0020

	NIM_ADD     = 0
	NIM_MODIFY  = 1
	NIM_DELETE  = 2
	NIF_MESSAGE = 0x1
	NIF_ICON    = 0x2
	NIF_TIP     = 0x4

	DWMWA_EXTENDED_FRAME_BOUNDS = 9
	DWMWA_CLOAKED               = 14

	WDA_NONE               = 0x0
	WDA_EXCLUDEFROMCAPTURE = 0x11

	MONITOR_DEFAULTTONEAREST = 2

	MB_OK              = 0x0
	MB_ICONINFORMATION = 0x40

	ERROR_ALREADY_EXISTS = 183

	HKEY_CURRENT_USER = 0x80000001
	KEY_QUERY_VALUE   = 0x0001
	KEY_SET_VALUE     = 0x0002
	REG_SZ            = 1

	QUNS_BUSY                    = 2
	QUNS_RUNNING_D3D_FULL_SCREEN = 3
	QUNS_PRESENTATION_MODE       = 4
)

type POINT struct{ X, Y int32 }
type SIZE struct{ CX, CY int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
	Private uint32
}

type WNDCLASSEX struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type BITMAPINFO struct {
	Header BITMAPINFOHEADER
	Colors [1]uint32
}

type BLENDFUNCTION struct {
	BlendOp, BlendFlags, SourceConstantAlpha, AlphaFormat byte
}

type MONITORINFO struct {
	Size    uint32
	Monitor RECT
	Work    RECT
	Flags   uint32
}

type ICONINFO struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

// NOTIFYICONDATA is the full Vista+ layout of NOTIFYICONDATAW (976 bytes on 64-bit Windows).
type NOTIFYICONDATA struct {
	Size            uint32
	Wnd             uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Version         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GuidItem        [16]byte
	BalloonIcon     uintptr
}

// Str converts a Go string to a NUL-terminated UTF-16 pointer.
func Str(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func ptr[T any](v *T) uintptr { return uintptr(unsafe.Pointer(v)) }

func RegisterClassEx(wc *WNDCLASSEX) uint16 {
	wc.Size = uint32(unsafe.Sizeof(*wc))
	r, _, _ := pRegisterClassExW.Call(ptr(wc))
	return uint16(r)
}

func CreateWindowEx(exStyle uint32, class, title string, style uint32, x, y, w, h int32, parent, instance uintptr) uintptr {
	r, _, _ := pCreateWindowExW.Call(uintptr(exStyle), uintptr(unsafe.Pointer(Str(class))), uintptr(unsafe.Pointer(Str(title))),
		uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), parent, 0, instance, 0)
	return r
}

func DefWindowProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	r, _, _ := pDefWindowProcW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

func DestroyWindow(hwnd uintptr) { pDestroyWindow.Call(hwnd) }

func PeekMessage(m *MSG) bool {
	r, _, _ := pPeekMessageW.Call(ptr(m), 0, 0, 0, PM_REMOVE)
	return r != 0
}

func TranslateMessage(m *MSG)  { pTranslateMessage.Call(ptr(m)) }
func DispatchMessage(m *MSG)   { pDispatchMessageW.Call(ptr(m)) }
func PostQuitMessage(code int) { pPostQuitMessage.Call(uintptr(code)) }
func PostMessage(hwnd uintptr, msg uint32, wp, lp uintptr) {
	pPostMessageW.Call(hwnd, uintptr(msg), wp, lp)
}

// MsgWait waits until a message arrives, the handle (if any) is signalled, or the timeout in milliseconds passes.
func MsgWait(handle uintptr, timeoutMs uint32) {
	if handle != 0 {
		h := handle
		pMsgWaitForMultipleObjectsEx.Call(1, ptr(&h), uintptr(timeoutMs), QS_ALLINPUT, MWMO_INPUTAVAILABLE)
	} else {
		pMsgWaitForMultipleObjectsEx.Call(0, 0, uintptr(timeoutMs), QS_ALLINPUT, MWMO_INPUTAVAILABLE)
	}
}

func GetCursorPos() (POINT, bool) {
	var p POINT
	r, _, _ := pGetCursorPos.Call(ptr(&p))
	return p, r != 0
}

func EnumWindows(cb uintptr) { pEnumWindows.Call(cb, 0) }

func IsWindowVisible(h uintptr) bool { r, _, _ := pIsWindowVisible.Call(h); return r != 0 }
func IsIconic(h uintptr) bool        { r, _, _ := pIsIconic.Call(h); return r != 0 }
func IsZoomed(h uintptr) bool        { r, _, _ := pIsZoomed.Call(h); return r != 0 }

func GetWindowLong(h uintptr, index uintptr) uintptr {
	r, _, _ := pGetWindowLongPtrW.Call(h, index)
	return r
}

func GetClassName(h uintptr) string {
	var buf [128]uint16
	n, _, _ := pGetClassNameW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:n])
}

// WindowFrame is the visible frame of a window: DWM's extended frame bounds, which leave out the invisible
// resize border, or the plain window rectangle if DWM can't tell.
func WindowFrame(h uintptr) (RECT, bool) {
	var r RECT
	hr, _, _ := pDwmGetWindowAttribute.Call(h, DWMWA_EXTENDED_FRAME_BOUNDS, ptr(&r), unsafe.Sizeof(r))
	if hr == 0 {
		return r, true
	}
	ok, _, _ := pGetWindowRect.Call(h, ptr(&r))
	return r, ok != 0
}

// IsCloaked reports whether DWM hides the window, as it does for windows on other virtual desktops and
// suspended store apps.
func IsCloaked(h uintptr) bool {
	var v uint32
	hr, _, _ := pDwmGetWindowAttribute.Call(h, DWMWA_CLOAKED, ptr(&v), 4)
	return hr == 0 && v != 0
}

// Monitors returns the full and work rectangles of each display.
func Monitors(cb uintptr) { pEnumDisplayMonitors.Call(0, 0, cb, 0) }

func GetMonitorInfo(hmon uintptr) (MONITORINFO, bool) {
	var mi MONITORINFO
	mi.Size = uint32(unsafe.Sizeof(mi))
	r, _, _ := pGetMonitorInfoW.Call(hmon, ptr(&mi))
	return mi, r != 0
}

func MonitorFromWindow(h uintptr) uintptr {
	r, _, _ := pMonitorFromWindow.Call(h, MONITOR_DEFAULTTONEAREST)
	return r
}

func GetSystemMetrics(i int) int32 {
	r, _, _ := pGetSystemMetrics.Call(uintptr(i))
	return int32(r)
}

func UpdateLayeredWindow(hwnd, hdcSrc uintptr, pos POINT, size SIZE) bool {
	src := POINT{}
	blend := BLENDFUNCTION{AC_SRC_OVER, 0, 255, AC_SRC_ALPHA}
	r, _, _ := pUpdateLayeredWindow.Call(hwnd, 0, ptr(&pos), ptr(&size), hdcSrc, ptr(&src), 0, ptr(&blend), ULW_ALPHA)
	return r != 0
}

func SetWindowPos(hwnd, after uintptr, x, y, w, h int32, flags uint32) {
	pSetWindowPos.Call(hwnd, after, uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(flags))
}

func ShowWindow(hwnd uintptr, cmd int) { pShowWindow.Call(hwnd, uintptr(cmd)) }

func GetDC(hwnd uintptr) uintptr { r, _, _ := pGetDC.Call(hwnd); return r }
func ReleaseDC(hwnd, dc uintptr) { pReleaseDC.Call(hwnd, dc) }
func CreateCompatibleDC(dc uintptr) uintptr {
	r, _, _ := pCreateCompatibleDC.Call(dc)
	return r
}
func SelectObject(dc, obj uintptr) uintptr { r, _, _ := pSelectObject.Call(dc, obj); return r }
func DeleteObject(obj uintptr)             { pDeleteObject.Call(obj) }
func DeleteDC(dc uintptr)                  { pDeleteDC.Call(dc) }

// CreateDIB makes a top-down 32-bit bitmap and returns it with its pixels.
func CreateDIB(w, h int) (uintptr, []uint32) {
	var bi BITMAPINFO
	bi.Header.Size = uint32(unsafe.Sizeof(bi.Header))
	bi.Header.Width = int32(w)
	bi.Header.Height = -int32(h)
	bi.Header.Planes = 1
	bi.Header.BitCount = 32
	var bits unsafe.Pointer
	bmp, _, _ := pCreateDIBSection.Call(0, ptr(&bi), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		return 0, nil
	}
	return bmp, unsafe.Slice((*uint32)(bits), w*h)
}

// CreateIcon makes an icon from premultiplied BGRA pixels.
func CreateIcon(pix []uint32, size int) uintptr {
	bmp, dst := CreateDIB(size, size)
	if bmp == 0 {
		return 0
	}
	copy(dst, pix)
	mask, _, _ := pCreateBitmap.Call(uintptr(size), uintptr(size), 1, 1, 0)
	ii := ICONINFO{FIcon: 1, HbmMask: mask, HbmColor: bmp}
	icon, _, _ := pCreateIconIndirect.Call(ptr(&ii))
	DeleteObject(bmp)
	DeleteObject(mask)
	return icon
}

func DestroyIcon(h uintptr) { pDestroyIcon.Call(h) }

func CreatePopupMenu() uintptr { r, _, _ := pCreatePopupMenu.Call(); return r }
func AppendMenu(menu uintptr, flags uint32, id uintptr, text string) {
	var t uintptr
	if text != "" {
		t = uintptr(unsafe.Pointer(Str(text)))
	}
	pAppendMenuW.Call(menu, uintptr(flags), id, t)
}
func TrackPopupMenu(menu uintptr, flags uint32, x, y int32, hwnd uintptr) uintptr {
	r, _, _ := pTrackPopupMenu.Call(menu, uintptr(flags), uintptr(x), uintptr(y), 0, hwnd, 0)
	return r
}
func DestroyMenu(menu uintptr)      { pDestroyMenu.Call(menu) }
func SetForegroundWindow(h uintptr) { pSetForegroundWindow.Call(h) }
func RegisterWindowMessage(name string) uint32 {
	r, _, _ := pRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(Str(name))))
	return uint32(r)
}

// SetExcludeFromCapture keeps the window out of screenshots and screen sharing (Windows 10 2004 and later).
func SetExcludeFromCapture(hwnd uintptr, exclude bool) {
	if pSetWindowDisplayAffinity.Find() != nil {
		return
	}
	v := uintptr(WDA_NONE)
	if exclude {
		v = WDA_EXCLUDEFROMCAPTURE
	}
	pSetWindowDisplayAffinity.Call(hwnd, v)
}

// SetDPIAware makes the process see physical pixels on every monitor.
func SetDPIAware() {
	if pSetProcessDpiAwarenessContext.Find() == nil {
		// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2
		if r, _, _ := pSetProcessDpiAwarenessContext.Call(^uintptr(3)); r != 0 {
			return
		}
	}
	if pSetProcessDPIAware.Find() == nil {
		pSetProcessDPIAware.Call()
	}
}

// SystemScale is the primary display's scale factor, 1 at 96 dpi.
func SystemScale() float32 {
	if pGetDpiForSystem.Find() != nil {
		return 1
	}
	r, _, _ := pGetDpiForSystem.Call()
	if r == 0 {
		return 1
	}
	return float32(r) / 96
}

func MessageBox(hwnd uintptr, text, caption string) {
	pMessageBoxW.Call(hwnd, uintptr(unsafe.Pointer(Str(text))), uintptr(unsafe.Pointer(Str(caption))), MB_OK|MB_ICONINFORMATION)
}

func LoadArrowCursor() uintptr {
	r, _, _ := pLoadCursorW.Call(0, 32512) // IDC_ARROW
	return r
}

func SetTimer(hwnd, id uintptr, ms uint32) { pSetTimer.Call(hwnd, id, uintptr(ms), 0) }
func KillTimer(hwnd, id uintptr)           { pKillTimer.Call(hwnd, id) }

func GetModuleHandle() uintptr { r, _, _ := pGetModuleHandleW.Call(0); return r }

// ExePath is the full path of the running executable.
func ExePath() string {
	var buf [1024]uint16
	n, _, _ := pGetModuleFileNameW.Call(0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:n])
}

// SingleInstance holds a named mutex and reports false if another copy already does.
func SingleInstance(name string) bool {
	h, _, err := pCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(Str(name))))
	if h == 0 {
		return true
	}
	if errno, ok := err.(syscall.Errno); ok && errno == ERROR_ALREADY_EXISTS {
		return false
	}
	return true
}

// NewFrameTimer makes a high-resolution waitable timer if the system has one (Windows 10 1803 and later),
// falling back to an ordinary one.
func NewFrameTimer() uintptr {
	const highRes = 0x2
	const access = 0x1F0003 // TIMER_ALL_ACCESS
	if pCreateWaitableTimerExW.Find() != nil {
		return 0
	}
	h, _, _ := pCreateWaitableTimerExW.Call(0, 0, highRes, access)
	if h == 0 {
		h, _, _ = pCreateWaitableTimerExW.Call(0, 0, 0, access)
	}
	return h
}

// ArmTimer sets the timer to fire once after the given number of 100-nanosecond intervals.
func ArmTimer(h uintptr, hundredNs int64) {
	due := -hundredNs
	pSetWaitableTimer.Call(h, ptr(&due), 0, 0, 0, 0)
}

func UILanguage() uint16 {
	r, _, _ := pGetUserDefaultUILanguage.Call()
	return uint16(r)
}

// NotifyIcon adds, changes or removes a tray icon.
func NotifyIcon(op uint32, d *NOTIFYICONDATA) bool {
	d.Size = uint32(unsafe.Sizeof(*d))
	r, _, _ := pShellNotifyIconW.Call(uintptr(op), ptr(d))
	return r != 0
}

// ShellBusy reports whether a full-screen app, game or presentation is in front.
func ShellBusy() bool {
	if pSHQueryUserNotificationState.Find() != nil {
		return false
	}
	var state int32
	hr, _, _ := pSHQueryUserNotificationState.Call(ptr(&state))
	if hr != 0 {
		return false
	}
	return state == QUNS_BUSY || state == QUNS_RUNNING_D3D_FULL_SCREEN || state == QUNS_PRESENTATION_MODE
}

func RegisterSessionNotification(hwnd uintptr) {
	if pWTSRegisterSessionNotification.Find() == nil {
		pWTSRegisterSessionNotification.Call(hwnd, 0) // NOTIFY_FOR_THIS_SESSION
	}
}

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func openRun(access uintptr) (uintptr, bool) {
	var key uintptr
	r, _, _ := pRegOpenKeyExW.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(Str(runKey))), 0, access, ptr(&key))
	return key, r == 0
}

// RunAtLogin reports whether the Run key has a value of this name.
func RunAtLogin(name string) bool {
	key, ok := openRun(KEY_QUERY_VALUE)
	if !ok {
		return false
	}
	defer pRegCloseKey.Call(key)
	r, _, _ := pRegQueryValueExW.Call(key, uintptr(unsafe.Pointer(Str(name))), 0, 0, 0, 0)
	return r == 0
}

// SetRunAtLogin adds or removes the command from the Run key.
func SetRunAtLogin(name, command string, on bool) bool {
	key, ok := openRun(KEY_SET_VALUE)
	if !ok {
		return false
	}
	defer pRegCloseKey.Call(key)
	if !on {
		r, _, _ := pRegDeleteValueW.Call(key, uintptr(unsafe.Pointer(Str(name))))
		return r == 0
	}
	data, _ := syscall.UTF16FromString(command)
	r, _, _ := pRegSetValueExW.Call(key, uintptr(unsafe.Pointer(Str(name))), 0, REG_SZ,
		uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)*2))
	return r == 0
}
