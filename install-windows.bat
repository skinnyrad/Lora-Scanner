@echo off

:: Function to check if Python3 is installed
python --version >nul 2>&1
IF %ERRORLEVEL% EQU 0 (
    echo Python3 is already installed.
    GOTO INSTALL_PACKAGES
)

echo Installing Python 3.12...
:: Download the most recent Python 3.12 installer
powershell -Command "Invoke-WebRequest -Uri https://www.python.org/ftp/python/3.12.0/python-3.12.0-amd64.exe -OutFile python-installer.exe"
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

:: Create a requirements.txt file
echo beautifulsoup4==4.12.3 > requirements.txt
echo Flask==2.3.2 >> requirements.txt
echo Flask_SocketIO==5.3.6 >> requirements.txt
echo MarkupSafe==2.1.3 >> requirements.txt
echo pyserial==3.5 >> requirements.txt
echo Requests==2.31.0 >> requirements.txt

:: Install required Python packages
pip install -r requirements.txt

:: Inform the user that the installation is complete
echo Installation complete. You can now run your application using: python app.py
pause
