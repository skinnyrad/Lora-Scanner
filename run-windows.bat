@echo off

:: Check if Python3 is installed
python --version >nul 2>&1
IF %ERRORLEVEL% EQU 0 (
    echo Running application with Python...
    python app.py
    GOTO END
)

python3 --version >nul 2>&1
IF %ERRORLEVEL% EQU 0 (
    echo Running application with Python3...
    python3 app.py
    GOTO END
)

echo Python is not installed. Please install Python3 to run the application.

:END
pause