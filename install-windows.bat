@echo off

:: Function to check if Python3 is installed
python --version >nul 2>&1
IF %ERRORLEVEL% EQU 0 (
    echo Python3 is already installed.
    GOTO INSTALL_PACKAGES
)

echo Installing Python 3.12...
:: Download the most recent Python 3.12 installer
powershell -Command "Invoke-WebRequest -Uri https://www.python.org/ftp/python/3.12.4/python-3.12.4-amd64.exe -OutFile python-installer.exe"
:: Install Python silently
python-installer.exe /quiet InstallAllUsers=1 PrependPath=1
:: Clean up installer
del python-installer.exe

:INSTALL_PACKAGES
:: Check if pip is installed
pip --version >nul 2>&1
IF %ERRORLEVEL% NEQ 0 (
    echo Installing pip...
    python -m ensurepip --upgrade
)

:: Install required Python packages
pip install flask flask-socketio pyserial beautifulsoup4 requests

:: Inform the user that the installation is complete
echo Installation complete. You can now run your application using: python your_script.py
pause
