# Skinny R&D Lora Scanner 

![lora-main](./doc/img/lora-main.png)

## Introduction
The Lora Scanner is a Flask web application integrated with SocketIO, designed for monitoring and analyzing LoRa (Long Range) wireless communication.

## Features

* **Packet Streaming**: View a live stream of each raw LoRa packet captured and received by the LoRa Scanner with `Analysis Mode`.
* **Supported Frequencies**: Capture packets at 915, 868, or 433 MHz
* **Device Tracking**: Track down LoRa Devices in the area using `Tracking Mode`.
* **Packet Analysis**: View every packet captured by the LoRa scanner with `Survey Mode`.
* **Exporting Data**: Perform AI powered data analysis on LoRa packets by downloading them from the `Survey Mode` page.
* **LoRaWAN Support**: Integrated LoRaWAN support with the Dragino LPS8N Indoor LoRaWAN Gateway.
* **Hardware Support**: The LoRa Scanner is currently designed to work with an `Adafruit Feather M0` or `32u4`.

## Installation

### Hardware Requirements

Flash your Adafruit Feather M0 (or 32u4) with the Skinny LoRa firmware.  You can attach up to three different receivers at a time.  For instructions, please follow the steps 1-2 [here](https://github.com/skinnyrad/Skinny-LoRa). 

*Note: For 433 MHz analysis you need the `Adafruit Feather M0 LoRa 433 MHz`*

### Prerequisites
- Python 3.x
- pip (Python package manager)
- Virtual environment (recommended)

### Setup For Linux, MacOS, and Windows 
1. Navigate to the application directory:
   ```
   cd lora-scanner
   ```
2. Create a virtual environment (optional):
   ```
   python -m venv venv
   ```
3. Activate the virtual environment (optional):

- Windows: `venv\Scripts\activate`
- macOS/Linux: `source venv/bin/activate`

4. Install the required packages:
   ```
   pip install -r requirements.txt
   ```

## Usage

1. Run the application:
```bash
python app.py
```
2. Access the web interface at `http://localhost:5000`.

![scanner-main](./doc/img/lora-main.png)


## Analysis Mode

Analyze LoRa traffic received at 433, 868, or 915 MHz with ‘Analysis Mode’. Click the desired frequency to get started. Once you are on the appropriate page, click the 'Connect Serial Port' button to connect to a serial port on your computer (the one your Feather is attached to). Once connected to your LoRa receiver, traffic will automatically be streamed to the web page for analysis. To disconnect a receiver, click the 'Disconnect Serial Port' button.

![analysis-mode](./doc/img/analysis-mode.png)

![connect-main](./doc/img/connect-main.png)

![connect-port](./doc/img/specify-port.png)

You can view and inspect any received packets from the analysis window in the appropriate section:

![analysis-mode-2](./doc/img/analysis-mode-2.png)

From analysis mode you can also transmit a message at the desired frequency using the ‘Transmit Data’ button.  

![transmit](./doc/img/transmit.png)

## Tracking Mode

Tracking mode allows you to track down rogue LoRa transmitters by locking in on a particular LoRa device and receiving live updates of their RSSI values.  From the main tracking mode screen, click the row containing the LoRa transmitter you are searching for and it will lock in on that particular device for seamless tracking.

![tracking-mode](./doc/img/tracking-mode.png)

If you toggle the beacon buttons at the bottom of the screen, you will find a similar interface to the ‘Analysis Mode’ page.  From this window you will be able to see live traffic received from the LoRa transceivers you have attached to the scanner.  If you would like to beacon a message at a desired frequency, you can use the ‘Start Beacon’ and ‘Stop Beacon’ buttons.  This will allow you to transmit a custom LoRa message at any desired interval , in an attempt to probe any LoRa devices that might be listening for messages:

![transmit-interval](./doc/img/transmit-interval.png)

## Survey Mode

Survey mode allows you to view all of the devices and corresponding packets discovered by the LoRa scanner.

![survey](./doc/img/survey-mode.png)

From the main ‘Survey Mode’ screen, you can view plaintext messages that were successfully decoded by the LoRa scanner using the ‘Show Values’ button next to a desired LoRa device.  After clicking the button, all of the messages captured from that particular device will be displayed.  Each row represents a single message captured from that device.

![survey-packets](./doc/img/survey-packets.png)

To collapse the packet list, click the ‘Hide Values’ button.

![survey-collapse](./doc/img/survey-collapse.png)

## Exporting LoRa Traffic

To download all packets captured by the LoRa Scanner, click the ‘Download Packets’ button.  This will export all packets into a CSV file.

![export-packets](./doc/img/download-packets.png)

## LoRaWAN

To capture LoRaWAN traffic, you have to connect to an active Dragino LPS8N Indoor LoRaWAN gateway (915 or 868 MHz).  You can connect to your gateway from the main screen by entering in the IP address of the gateway.  Currently the application allows you to simultaneously connect to three gateways at once.

![gateway](./doc/img/lorawan-gateway.png)

![gateway-config](./doc/img/lorawan-gateway-configure.png)
