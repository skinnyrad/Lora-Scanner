#!/bin/bash

# Function to check if Python3 is installed
check_python_installed() {
    if command -v python3 &> /dev/null; then
        echo "Python3 is already installed."
        return 0
    else
        return 1
    fi
}

# Install Python3 and pip if not already installed
if ! check_python_installed; then
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        sudo apt update
        sudo apt install -y python3 python3-pip
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        # Check if Homebrew is installed, install if not
        if ! command -v brew &> /dev/null; then
            echo "Homebrew not found, installing..."
            /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
        fi
        brew install python3
    fi
else
    echo "Skipping Python3 installation."
fi

# Install required Python packages
pip3 install flask flask-socketio pyserial beautifulsoup4 requests

# Inform the user that the installation is complete
echo "Installation complete. You can now run your application using: python3 your_script.py"

