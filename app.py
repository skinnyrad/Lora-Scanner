from flask import Flask, render_template, request, jsonify, send_file
from markupsafe import escape
from flask_socketio import SocketIO, emit
import serial
import threading
from threading import Timer
import time
from collections import deque
import re
import requests
from bs4 import BeautifulSoup
import ipaddress
from io import StringIO, BytesIO
import csv
import serial.tools.list_ports
from ipaddress import ip_address, AddressValueError

app = Flask(__name__)
socketio = SocketIO(app)
serial_threads = {}
serial_buffers = {}
port1_status = True
port2_status = True
port3_status = True
global ser1
global ser2 
global ser3
gateway_ips = {
    'gateway1': '',
    'gateway2': '',
    'gateway3': '',
    'gateway4': '',
    'gateway5': '',
    'gateway6': '',
    'gateway7': '',
    'gateway8': '',
    'gateway9': '',
    'gateway10': ''
}
frequency = lambda port: {'port1': 433, 'port2': 868,'port3': 915}.get(port, None)
surveydata = {}
parsed_entries = set()
used_ports = set()

def read_serial_data(port, ser, buffer):
    """
    This function reads data from a serial port and processes it.

    Parameters:
    port (str): The name of the serial port (e.g., 'port1', 'port2', 'port3').
    ser (Serial): The serial object representing the serial port.
    buffer (deque): A deque object to store the received data.

    Returns:
    None
    """
    global surveydata
    rssi_pattern = r"RSSI: (-?\d+)"
    decoded_value_pattern = r"Decoded Value: (.+)"
    rssi = None
    decoded_value = None

    while True:
        try:
            if ser.in_waiting > 0:
                data = ser.readline().decode('utf-8').strip()

                # Match RSSI
                rssi_match = re.search(rssi_pattern, data)
                if rssi_match:
                    rssi = int(rssi_match.group(1))

                # Match Decoded Value
                decoded_match = re.search(decoded_value_pattern, data)
                if decoded_match:
                    decoded_value = decoded_match.group(1)

                # Update dictionary only if both RSSI and Decoded Value are found
                if rssi is not None and decoded_value is not None:
                    freq = frequency(port)
                    key = f'Raw LoRa Device {freq} MHz'

                    # Initialize the list for the frequency if not already done
                    if key not in surveydata:
                        surveydata[key] = []

                    # Append the new values to the list for this frequency
                    current_timestamp = time.strftime('%Y-%m-%d %H:%M:%S', time.localtime())
                    surveydata[key].append([freq, rssi, f"{decoded_value} -- Timestamp:{current_timestamp}"])

                    # Reset rssi and decoded_value for next packet
                    rssi = None
                    decoded_value = None

                buffer.append(data)
                socketio.emit(f'serial_data_{port}', {'data': data})

            # Check port status
            if (port == 'port1' and not port1_status) or \
               (port == 'port2' and not port2_status) or \
               (port == 'port3' and not port3_status):
                return

            time.sleep(0.1)

        except Exception as e:
            #print("Could not access Serial Port")
            #print(f"Error: {e}")
            pass

def convert_dict_to_csv(data):
    """
    Converts a dictionary of survey data into a CSV format.

    Parameters:
    data (dict): A dictionary where keys are device identifiers and values are lists of data entries.
                 Each data entry is a list containing frequency, RSSI, and a decoded value.

    Returns:
    StringIO: A StringIO object containing the CSV representation of the input data.
    """
    output = StringIO()
    writer = csv.writer(output)

    # Write the header
    writer.writerow(['Device', 'Frequency', 'RSSI', 'Decoded Value'])

    # Write the data
    for key, values in data.items():
        for value in values:
            writer.writerow([key] + value)

    output.seek(0)
    return output

def is_valid_ip(ip):
    """
    Validates whether a given string is a valid IP address.

    Parameters:
    ip (str): The string representation of the IP address to validate.

    Returns:
    bool: True if the input string is a valid IP address, False otherwise.
    """
    try:
        ip_address(ip)
        return True
    except AddressValueError:
        return False

def parse_and_store_data():
    """
    Fetches data from configured gateway URLs, parses the data, and stores it in a global survey data structure.

    This function constructs URLs for gateways, sends HTTP GET requests to fetch data, parses the HTML response
    to extract relevant information, and stores the parsed data in a global dictionary. It also schedules itself
    to run periodically.

    Global Variables:
    - surveydata (dict): A dictionary to store parsed survey data.
    - parsed_entries (set): A set to keep track of already parsed entries to avoid duplicates.
    - gateway_ips (dict): A dictionary containing IP addresses of gateways.

    The function does not take any parameters or return any values.
    """
    global surveydata
    global parsed_entries
    global gateway_ips
    # Include the port number (8000) in your gateway URLs
    gateway_urls = [
        f"http://{gateway_ips[f'gateway{i}']}:8000/cgi-bin/log-traffic.has" 
        for i in range(1, 11) 
        if gateway_ips[f'gateway{i}'] and is_valid_ip(gateway_ips[f'gateway{i}'])
    ]

    headers = {
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:122.0) Gecko/20100101 Firefox/122.0",
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
        "Accept-Language": "en-US,en;q=0.5",
        "Accept-Encoding": "gzip, deflate",
        "DNT": "1",
        "Sec-GPC": "1",
        "Authorization": "Basic cm9vdDpkcmFnaW5v",  # Assumes the same credentials for both gateways
        "Connection": "keep-alive",
        "Upgrade-Insecure-Requests": "1"
    }

    for url in gateway_urls:
        try:
            print(f"Fetching data from {url}")
            response = requests.get(url, headers=headers, timeout=10)

            if response.status_code == 200:
                soup = BeautifulSoup(response.text, 'html.parser')
                table = soup.find('table')
                if table:
                    rows = table.find_all('tr')[1:]  # Skip the header row

                    for row in rows:
                        # Skip hidden rows in this iteration
                        if row.get('style') == 'display: none;':
                            continue

                        cells = row.find_all('td')
                        if not cells:
                            continue

                        # Prepare formatted_row from visible cells, skipping the first cell for Chevron icon
                        formatted_row = ' | '.join(cell.text.strip() for cell in cells[1:])

                        # Extract dev_id and freq from the visible row
                        dev_id = extract_dev_id(formatted_row)
                        freq = extract_freq(formatted_row)

                        # Extract RSSI from the next hidden row
                        hidden_row = row.find_next_sibling('tr')
                        if hidden_row and 'display: none;' in hidden_row.get('style', ''):
                            hidden_data = hidden_row.td.text.strip()
                            rssi_match = re.search(r'"Rssi":(-?\d+)', hidden_data)
                            if rssi_match:
                                rssi = int(rssi_match.group(1))
                            else:
                                rssi = None
                        else:
                            rssi = None

                        # Save data into the surveydata structure
                        if dev_id and freq is not None:
                            entry_identifier = f"{dev_id}_{freq}_{formatted_row}"
                            if entry_identifier not in parsed_entries:
                                parsed_entries.add(entry_identifier)
                                if dev_id not in surveydata:
                                    surveydata[dev_id] = []
                                surveydata[dev_id].append([freq, rssi, formatted_row])

                        print(f"Data parsed and stored from {url}.")
            else:
                print(f"Request to {url} failed with status code: {response.status_code}")
        except Exception as e:
            print(f"An error occurred while processing {url}: {e}")
    # Schedule the next call to this function
    Timer(30, parse_and_store_data).start()  # Call this function every 30 seconds



def extract_dev_id(formatted_row):
    """
    Extracts the device identifier (DevEui or DevAddr) from a formatted row string.

    The function assumes that the device identifier is located in the 'Content' part of the formatted row,
    formatted as 'Dev Addr: {DevEui}, Size: {Size}'.

    Parameters:
    formatted_row (str): A string representing a row of data, where the last part contains the 'Content'
                         with the device identifier.

    Returns:
    str or None: The extracted device identifier (DevEui or DevAddr) if successful, or None if extraction fails.
    """
    try:
        content_part = formatted_row.split('|')[-1].strip()  # Get the last part of the formatted_row, which is 'Content'
        dev_id = content_part.split(',')[0].split(':')[-1].strip()  # Extract the DevEui or DevAddr
        return dev_id
    except Exception as e:
        print(f"Error extracting DevEui/DevAddr: {e}")
        return None  # Return None or some default value if extraction fails


def extract_freq(formatted_row):
    """
    Extracts the frequency value from a formatted row string.

    The function assumes that the frequency is a standalone field within the formatted row,
    specifically located at a fixed position.

    Parameters:
    formatted_row (str): A string representing a row of data, where fields are separated by '|'.

    Returns:
    float or None: The extracted frequency as a float if successful, or None if extraction fails.
    """
    try:
        freq_part = formatted_row.split('|')[3].strip()  # Get the 'Freq' part (assuming it's the fifth field)
        freq = float(freq_part)  # Convert the frequency to float
        return freq
    except Exception as e:
        print(f"Error extracting frequency: {e}")
        return None  # Return None or some default value if extraction fails


def connect_serial(port, frequency):
    """
    Establishes a serial connection to a specified port and frequency, and starts a thread to read data.

    Parameters:
    port (str): The name of the serial port to connect to.
    frequency (int): The frequency in MHz for which the serial connection is being established. 
                     Supported frequencies are 433, 868, and 915 MHz.

    Returns:
    None
    """
    global ser1
    global ser2 
    global ser3
    global used_ports

    # Check if the port is already in use
    if port in used_ports:
        raise Exception(f"Port {port} is already in use for another frequency.")

    if frequency == 433:
        try:
            ser1 = serial.Serial(port, baudrate=9600)
            serial_buffers['port1'] = deque(maxlen=10)
            serial_threads['port1'] = threading.Thread(target=read_serial_data, args=('port1', ser1, serial_buffers['port1']))
            serial_threads['port1'].daemon = True
            serial_threads['port1'].start()
            used_ports.add(port)  # Mark the port as used
        except:
            print("\n\nPort for 433 MHz not available\n\n")
    if frequency == 868:
        try:
            ser2 = serial.Serial(port, baudrate=9600)
            serial_buffers['port2'] = deque(maxlen=10)
            serial_threads['port2'] = threading.Thread(target=read_serial_data, args=('port2', ser2, serial_buffers['port2']))
            serial_threads['port2'].daemon = True
            serial_threads['port2'].start()
            used_ports.add(port)  # Mark the port as used
        except:
            print("\n\nPort for 868 MHz not available\n\n")
    if frequency == 915:
        try:
            ser3 = serial.Serial(port, baudrate=9600)
            serial_buffers['port3'] = deque(maxlen=10)
            serial_threads['port3'] = threading.Thread(target=read_serial_data, args=('port3', ser3, serial_buffers['port3']))
            serial_threads['port3'].daemon = True
            serial_threads['port3'].start()
            used_ports.add(port)  # Mark the port as used
        except:
            print('\n\nPort for 915 MHz not available\n\n')


def disconnect_serial(port):
    """
    Disconnects the serial connection for a specified port and cleans up associated resources.

    Parameters:
    port (str): The name of the serial port to disconnect. Expected values are 'port1', 'port2', or 'port3'.

    Returns:
    None
    """
    global ser1
    global ser2 
    global ser3
    global used_ports
    try:
        serial_threads[port].stop()
        del serial_threads[port]
        del serial_buffers[port]
    except:
        pass
    if port == "port1":
        ser1.close()
        used_ports.remove(ser1.port)  # Remove the port from the used set
    elif port == "port2":
        ser2.close()
        used_ports.remove(ser2.port)  # Remove the port from the used set
    elif port == "port3":
        ser3.close()
        used_ports.remove(ser3.port)  # Remove the port from the used set
    else:
        print("Unknown port, something went wrong...")
    

@app.route('/')
def index():
    """
    Renders the main index page of the application.

    This function handles the root URL of the application and returns the rendered HTML
    template for the index page.

    Returns:
    str: The rendered HTML content for the index page.
    """
    return render_template('index.html', gateway_ips=gateway_ips)

@app.route('/analysis')
def analysis():
    """
    Renders the analysis page of the application with initial serial data.

    This function handles the '/analysis' URL route and returns the rendered HTML
    template for the analysis page. It includes initial data from the serial buffers
    for each port.

    Returns:
    str: The rendered HTML content for the analysis page, with initial data for each port.
    """
    return render_template('analysis.html', initial_data={port: list(buffer) for port, buffer in serial_buffers.items()})


@app.route('/survey')
def survey():
    """
    Renders the survey page of the application.

    This function handles the '/survey' URL route and returns the rendered HTML
    template for the survey page.

    Returns:
    str: The rendered HTML content for the survey page.
    """
    return render_template('survey.html')

@app.route('/tracking')
def tracking():
    """
    Renders the tracking page of the application with initial serial data.

    This function handles the '/tracking' URL route and returns the rendered HTML
    template for the tracking page. It includes initial data from the serial buffers
    for each port.

    Returns:
    str: The rendered HTML content for the tracking page, with initial data for each port.
    """
    return render_template('tracking.html', initial_data={port: list(buffer) for port, buffer in serial_buffers.items()})

@app.route('/attach_serial_433', methods=['GET'])
def attach_serial_433():
    """
    Attaches a serial connection to the 433 MHz port.

    This function handles the '/attach_serial_433' URL route and establishes a serial
    connection to the specified port for 433 MHz frequency.

    Parameters:
    None

    Returns:
    Response: A JSON response indicating the result of the operation.
    """
    try:
        # Disconnect any existing connection for 433 MHz
        disconnect_serial("port1")
        print("Current port for 433 MHz disconnected")
    except:
        pass
    
    user_input = escape(request.args.get('user_input'))
    connect_serial(str(user_input), 433)
    result = f'Serial Port Requested for 433 MHz'
    return jsonify(result=result)

@app.route('/delete_serial_433', methods=['GET'])
def delete433():
    """
    Disconnects the serial connection for the 433 MHz port.

    This function handles the '/delete_serial_433' URL route and disconnects
    the serial connection associated with the 433 MHz frequency.

    Returns:
    Response: A JSON response indicating the result of the disconnection operation.
    """
    disconnect_serial("port1")
    result = "Port Disconnected!"
    return jsonify(result=result)

@app.route('/attach_serial_868', methods=['GET'])
def attach_serial_868():
    """
    Attaches a serial connection to the 868 MHz port.

    This function handles the '/attach_serial_868' URL route and establishes a serial
    connection to the specified port for 868 MHz frequency.

    Parameters:
    None

    Returns:
    Response: A JSON response indicating the result of the operation.
    """
    try:
        # Disconnect any existing connection for 868 MHz
        disconnect_serial("port2")
        print("Current port for 868 MHz disconnected")
    except:
        pass
    
    user_input = escape(request.args.get('user_input'))
    connect_serial(str(user_input), 868)
    result = f'Serial Port Requested for 868 MHz'
    return jsonify(result=result)

@app.route('/delete_serial_868', methods=['GET'])
def delete868():
    """
    Disconnects the serial connection for the 868 MHz port.

    This function handles the '/delete_serial_868' URL route and disconnects
    the serial connection associated with the 868 MHz frequency.

    Returns:
    Response: A JSON response indicating the result of the disconnection operation.
    """
    disconnect_serial("port2")
    result = "Port Disconnected!"
    return jsonify(result=result)

@app.route('/attach_serial_915', methods=['GET'])
def attach_serial_915():
    """
    Attaches a serial connection to the 915 MHz port.

    This function handles the '/attach_serial_915' URL route and establishes a serial
    connection to the specified port for 915 MHz frequency.

    Parameters:
    None

    Returns:
    Response: A JSON response indicating the result of the operation.
    """
    try:
    # Disconnect any existing connection for 915 MHz
        disconnect_serial("port3")
        print("Current port for 915 MHz disconnected")
    except:
        pass
    
    user_input = escape(request.args.get('user_input'))
    connect_serial(str(user_input), 915)
    result = f'Serial Port Requested for 915 MHz'
    return jsonify(result=result)

@app.route('/delete_serial_915', methods=['GET'])
def delete915():
    """
    Disconnects the serial connection for the 915 MHz port.

    This function handles the '/delete_serial_915' URL route and disconnects
    the serial connection associated with the 915 MHz frequency.

    Returns:
    Response: A JSON response indicating the result of the disconnection operation.
    """
    disconnect_serial("port3")
    result = "Port Disconnected!"
    return jsonify(result=result)

@socketio.on('connect')
def handle_connect():
    """
    Handles a new client connection to the SocketIO server.

    This function is triggered when a client connects to the server via SocketIO.
    It emits the initial serial data for each port to the connected client.

    Parameters:
    None

    Returns:
    None
    """
    for port, buffer in serial_buffers.items():
        emit(f'initial_serial_data_{port}', {'data': list(buffer)})

@app.route('/transmit433', methods=['POST'])
def transmit433():
    """
    Transmits a message over the 433 MHz serial connection.

    This function handles the '/transmit433' URL route and sends a user-provided
    message to the connected serial device operating at 433 MHz frequency.

    Parameters:
    None

    Returns:
    Response: A JSON response indicating the result of the transmission operation.
    """
    global ser1 
    data = request.json  # Get the data from the POST request
    user_input = data.get('user_input')  # Extract the user input
    msg = "TX:"+user_input+" "
    ser1.write(msg.encode())
    return jsonify(result="Ok")

@app.route('/get_serial_ports')
def get_serial_ports():
    """
    Retrieves a list of available serial ports on the system.

    This function handles the '/get_serial_ports' URL route and returns a JSON
    response containing a list of serial port device names available on the system.

    Returns:
    Response: A JSON response with a list of available serial port device names.
    """
    ports = [port.device for port in serial.tools.list_ports.comports()]
    return jsonify(ports=ports)

@app.route('/transmit868', methods=['POST'])
def transmit868():
    """
    Transmits a message over the 868 MHz serial connection.

    This function handles the '/transmit868' URL route and sends a user-provided
    message to the connected serial device operating at 868 MHz frequency.

    Parameters:
    None

    Returns:
    Response: A JSON response indicating the result of the transmission operation.
    """
    global ser2 
    data = request.json  # Get the data from the POST request
    user_input = data.get('user_input')  # Extract the user input
    msg = "TX:"+user_input+" "
    ser2.write(msg.encode())
    return jsonify(result="Ok")

@app.route('/transmit915', methods=['POST'])
def transmit915():
    """
    Transmits a message over the 915 MHz serial connection.

    This function handles the '/transmit915' URL route and sends a user-provided
    message to the connected serial device operating at 915 MHz frequency.

    Parameters:
    None

    Returns:
    Response: A JSON response indicating the result of the transmission operation.
    """
    global ser3
    data = request.json  # Get the data from the POST request
    user_input = data.get('user_input')  # Extract the user input
    msg = "TX:"+user_input+" "
    ser3.write(msg.encode())
    return jsonify(result="Ok")

@app.route('/checkSer', methods=['GET'])
def checkSer():
    """
    Checks the status of a specified serial port to determine if it is open.

    This function handles the '/checkSer' URL route and checks whether the specified
    serial port is currently open. It returns a JSON response indicating the status.

    Parameters:
    None

    Returns:
    Response: A JSON response with the result "True" if the specified port is open,
              otherwise "False".
    """
    data = request.args.get('port')
    if data == 'port1':
        try:
            if ser1.is_open:
                return jsonify(result="True")
        except:
            pass
    elif data == 'port2':
        try:
            if ser2.is_open:
                return jsonify(result="True")
        except:
            pass
    elif data == 'port3':
        try:
            if ser3.is_open:
                return jsonify(result="True")
        except:
            pass
    return jsonify(result="False")

@app.route('/get_table_data')        
def get_table_data():
    """
    Retrieves and returns the cleaned survey data in JSON format.

    This function processes the global survey data to remove any entries with empty device identifiers
    and returns the cleaned data as a JSON response.

    Parameters:
    None

    Returns:
    Response: A JSON response containing the cleaned survey data, where each key is a device identifier
              and the value is a list of associated data entries.
    """
    global surveydata
    cleaned_data = {}

    for dev_id, data in surveydata.items():
        if dev_id:  # Check if dev_id is not empty
            cleaned_data[dev_id] = data

    return jsonify(cleaned_data)


@app.route('/set_gateways', methods=['POST'])
def set_gateways():
    """
    Updates the IP addresses of the gateways based on the provided form data.

    This function processes a POST request containing IP addresses for up to 10 gateways.
    It validates each IP address and updates the global `gateway_ips` dictionary with the
    new values if they are valid.

    Parameters:
    None

    Returns:
    Response: A JSON response indicating the success or failure of the operation.
              - On success, returns a message confirming the update of gateway IPs with a 200 status code.
              - On failure, returns an error message specifying the invalid IP address with a 400 status code.
    """
    global gateway_ips
    data = request.form
    
    # Create an ordered dictionary of gateways
    ordered_gateways = {}
    for i in range(1, 11):
        key = f'gateway{i}'
        input_ip = data.get(key, '').strip()
        ordered_gateways[key] = input_ip
        
    # Update the global gateway_ips with ordered data
    gateway_ips.update(ordered_gateways)

    for gateway, ip_address in gateway_ips.items():
        print(f"Gateway {gateway} has IP address: {ip_address}")

    return jsonify({"message": "Gateway IPs updated successfully"}), 200

@app.route('/get_gateways', methods=['GET'])
def get_gateways():
    return jsonify(gateway_ips)

@app.route('/downloadPackets', methods=['GET'])
def downloadPackets():
    """
    Initiates the download of survey data in CSV format.

    This function handles the '/downloadPackets' URL route and converts the global
    survey data into a CSV format. It then prepares the CSV data for download as a file.

    Parameters:
    None

    Returns:
    Response: A Flask response object that sends the CSV file as an attachment to the client.
              The file is named 'surveydata.csv' and is sent with a 'text/csv' MIME type.
    """
    csv_data = convert_dict_to_csv(surveydata)
    # Convert StringIO to BytesIO for send_file compatibility
    bytes_data = BytesIO()
    bytes_data.write(csv_data.getvalue().encode('utf-8'))
    bytes_data.seek(0)
    return send_file(bytes_data, mimetype='text/csv', as_attachment=True, download_name='surveydata.csv')



if __name__ == '__main__':
    Timer(30, parse_and_store_data).start()
    socketio.run(app, debug=True)
