@echo off
rem Builds dist\SnowForWindows.exe. Needs Python 3.9 or later from python.org.
python -m pip install -r requirements.txt pyinstaller || exit /b 1
python -m PyInstaller --noconfirm --onefile --windowed --name SnowForWindows --icon assets\snowflake.ico run.pyw || exit /b 1
echo.
echo Done: dist\SnowForWindows.exe
