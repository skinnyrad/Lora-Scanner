# Lora Scanner Application

## Introduction
The Lora Scanner is a Flask web application integrated with SocketIO, designed for real-time communication and management of serial data across various frequency bands. This application is ideal for monitoring and analyzing LoRa (Long Range) wireless communication, offering features like device tracking, data analysis, and survey mode.

## Features
- **Real-Time Data Processing**: Read and process data from serial ports in real-time.
- **Frequency Band Management**: Manage connections across 433 MHz, 868 MHz, and 915 MHz bands.
- **Interactive Web Pages**: Dedicated pages for dashboard, analysis, survey, and tracking.
- **WebSocket Integration**: Real-time data streaming with SocketIO.
- **Serial Data Transmission**: Transmit data over serial connections.

## Installation

### Prerequisites
- Python 3.x
- pip (Python package manager)
- Virtual environment (recommended)

### Setup
1. Navigate to the application directory:
   ```bash
   cd lora-scanner
   ```
2. Create a virtual environment (optional):
   ```bash
   python -m venv venv
   ```
3. Activate the virtual environment:
   - Windows: `venv\Scripts\activate`
   - macOS/Linux: `source venv/bin/activate`
4. Install the required packages:
   ```bash
   pip install -r requirements.txt
   ```

## Usage

1. Run the application:
   ```bash
   python app.py
   ```
2. Access the web interface at `http://localhost:5000`.

## API Endpoints
- `/` - Homepage
- `/analysis` - Analysis Page
- `/survey` - Survey Page
- `/tracking` - Tracking Page
- Additional endpoints for managing serial connections and data transmission.

