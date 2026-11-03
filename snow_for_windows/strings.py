"""The menu's words: Ukrainian for a Ukrainian Windows, English otherwise."""

from types import SimpleNamespace

ENGLISH = SimpleNamespace(
    app_name="Snow for Windows",
    tip="Snow for Windows",
    paused_tip="Snow for Windows — snowfall off",
    snowfall="Snowfall",
    levels=["Off", "Light", "Steady", "Heavy", "Blizzard", "Variable"],
    accumulate="Accumulate",
    melt="Melt Snow",
    on_windows="Snow on Windows",
    clear="Clear Snow",
    in_screenshots="Show in Screenshots",
    smooth="Smooth Motion (60 fps)",
    start_at_login="Start with Windows",
    about="About",
    quit="Quit",
    about_text=(
        "Snow for Windows\n\nSoft snow drifts across your screen, settles on the tops of your windows "
        "and slips off when you move them. Sweep the drifts aside with the cursor.\n\n"
        "Everything lives in the snowflake icon in the tray."
    ),
    welcome=(
        "It's snowing on your desktop. The snowflake icon by the clock holds all the settings: "
        "how hard it snows, melting, snow on windows, and Quit."
    ),
)

UKRAINIAN = SimpleNamespace(
    app_name="Snow for Windows",
    tip="Snow for Windows — сніг на робочому столі",
    paused_tip="Snow for Windows — снігопад вимкнено",
    snowfall="Снігопад",
    levels=["Вимкнено", "Легкий", "Помірний", "Сильний", "Хуртовина", "Мінливий"],
    accumulate="Накопичувати сніг",
    melt="Танення снігу",
    on_windows="Сніг на вікнах",
    clear="Прибрати сніг",
    in_screenshots="Показувати на скріншотах",
    smooth="Плавний рух (60 кадрів/с)",
    start_at_login="Запускати з Windows",
    about="Про програму",
    quit="Вийти",
    about_text=(
        "Snow for Windows\n\nМ'який сніг падає на екран, лягає на верхні краї вікон і зсипається, "
        "коли їх рухаєш. Розгрібай замети курсором.\n\n"
        "Усі налаштування — в іконці сніжинки в треї."
    ),
    welcome=(
        "На твоєму робочому столі йде сніг. Усі налаштування — в іконці сніжинки біля годинника: "
        "сила снігопаду, танення, сніг на вікнах і вихід."
    ),
)


def for_language(lang_id: int) -> SimpleNamespace:
    return UKRAINIAN if lang_id & 0x3FF == 0x22 else ENGLISH  # LANG_UKRAINIAN
