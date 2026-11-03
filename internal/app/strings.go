package app

// Text holds the menu's words in one language.
type Text struct {
	AppName, Tip                                   string
	Intensity, Off, Light, Steady, Heavy, Blizzard string
	Variable                                       string
	Accumulate, Melt, OnWindows, InScreenshot      string
	Smooth, StartAtLogin, Clear, About, Quit       string
	AboutText, Welcome                             string
	PausedTip                                      string
}

var english = Text{
	AppName:      "Snow for Windows",
	Tip:          "Snow for Windows",
	PausedTip:    "Snow for Windows — snowfall off",
	Intensity:    "Snowfall",
	Off:          "Off",
	Light:        "Light",
	Steady:       "Steady",
	Heavy:        "Heavy",
	Blizzard:     "Blizzard",
	Variable:     "Variable",
	Accumulate:   "Accumulate",
	Melt:         "Melt Snow",
	OnWindows:    "Snow on Windows",
	InScreenshot: "Show in Screenshots",
	Smooth:       "Smooth Motion (60 fps)",
	StartAtLogin: "Start with Windows",
	Clear:        "Clear Snow",
	About:        "About",
	Quit:         "Quit",
	AboutText: "Snow for Windows\n\nSoft snow drifts across your screen, settles on the tops of your windows " +
		"and slips off when you move them. Sweep the drifts aside with the cursor.\n\n" +
		"Everything lives in the snowflake icon in the tray.",
	Welcome: "It's snowing on your desktop.\n\nThe snowflake icon in the tray (by the clock) holds all the " +
		"settings: how hard it snows, melting, snow on windows, and Quit.",
}

var ukrainian = Text{
	AppName:      "Snow for Windows",
	Tip:          "Snow for Windows — сніг на робочому столі",
	PausedTip:    "Snow for Windows — снігопад вимкнено",
	Intensity:    "Снігопад",
	Off:          "Вимкнено",
	Light:        "Легкий",
	Steady:       "Помірний",
	Heavy:        "Сильний",
	Blizzard:     "Хуртовина",
	Variable:     "Мінливий",
	Accumulate:   "Накопичувати сніг",
	Melt:         "Танення снігу",
	OnWindows:    "Сніг на вікнах",
	InScreenshot: "Показувати на скріншотах",
	Smooth:       "Плавний рух (60 кадрів/с)",
	StartAtLogin: "Запускати з Windows",
	Clear:        "Прибрати сніг",
	About:        "Про програму",
	Quit:         "Вийти",
	AboutText: "Snow for Windows\n\nМ'який сніг падає на екран, лягає на верхні краї вікон і зсипається, " +
		"коли їх рухаєш. Розгрібай замети курсором.\n\n" +
		"Усі налаштування — в іконці сніжинки в треї.",
	Welcome: "На твоєму робочому столі йде сніг.\n\nУсі налаштування — в іконці сніжинки в треї " +
		"(біля годинника): сила снігопаду, танення, сніг на вікнах і вихід.",
}

// TextFor picks Ukrainian for a Ukrainian UI language and English otherwise.
func TextFor(langID uint16) *Text {
	if langID&0x3ff == 0x22 { // LANG_UKRAINIAN
		return &ukrainian
	}
	return &english
}
