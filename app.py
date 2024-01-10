from flask import Flask, render_template, request, jsonify
from markupsafe import escape
from flask_socketio import SocketIO, emit
import serial
import threading
import time
from collections import deque
import pandas as pd
import re

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
global_dataframe = pd.DataFrame(columns=['Device Name', 'Frequency', 'Signal Strength', 'Plaintext'])
frequency = lambda port: {'port1': 433, 'port2': 868,'port3': 915}.get(port, None)
surveydata = {}

def read_serial_data(port, ser, buffer):
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
                    surveydata[key].append([freq, rssi, decoded_value])

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
            print(f"Error: {e}")
            pass





def connect_serial(port,frequency):
    global ser1
    global ser2 
    global ser3

    if frequency == 433:
        try:
            ser1 = serial.Serial(port, baudrate=9600)
            serial_buffers['port1'] = deque(maxlen=10)
            serial_threads['port1'] = threading.Thread(target=read_serial_data, args=('port1', ser1, serial_buffers['port1']))
            serial_threads['port1'].daemon = True
            serial_threads['port1'].start()
        except:
            print("\n\nPort for 433 MHz not available\n\n")
    if frequency == 868:
        try:
            ser2 = serial.Serial(port, baudrate=9600)
            serial_buffers['port2'] = deque(maxlen=10)
            serial_threads['port2'] = threading.Thread(target=read_serial_data, args=('port2', ser2, serial_buffers['port2']))
            serial_threads['port2'].daemon = True
            serial_threads['port2'].start()
        except:
            print("\n\nPort for 868 MHz not available\n\n")
    if frequency == 915:
        try:
            ser3 = serial.Serial(port, baudrate=9600)
            serial_buffers['port3'] = deque(maxlen=10)
            serial_threads['port3'] = threading.Thread(target=read_serial_data, args=('port3', ser3, serial_buffers['port3']))
            serial_threads['port3'].daemon = True
            serial_threads['port3'].start()
            
        except:
            print('\n\nPort for 915 MHz not available\n\n')


def disconnect_serial(port):
    global ser1
    global ser2 
    global ser3
    try:
        serial_threads[port].stop()
        del serial_threads[port]
        del serial_buffers[port]
    except:
        pass
    if port == "port1":
        ser1.close()
    elif port == "port2":
        ser2.close()
    elif port == "port3":
        ser3.close()
    else:
        print("Unkown port, something went wrong...")
    

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/analysis')
def analysis():
    return render_template('analysis.html', initial_data={port: list(buffer) for port, buffer in serial_buffers.items()})

@app.route('/survey')
def survey():
    return render_template('survey.html', data=global_dataframe)

@app.route('/tracking')
def tracking():
    return render_template('tracking.html', initial_data={port: list(buffer) for port, buffer in serial_buffers.items()})

@app.route('/attach_serial_433', methods=['GET'])
def attach_serial_433():
    user_input = escape(request.args.get('user_input'))
    port1_status = True
    connect_serial(str(user_input), 433)
    # Process the input as needed
    result = f'Serial Port Requested for 433 MHz'
    return jsonify(result=result)

@app.route('/delete_serial_433', methods=['GET'])
def delete433():
    # Add your logic here to handle the confirmation
    disconnect_serial("port1")
    result = "Port Disconnected!"
    return jsonify(result=result)

@app.route('/attach_serial_868', methods=['GET'])
def attach_serial_868():
    user_input = escape(request.args.get('user_input'))
    port2_status = True
    connect_serial(str(user_input), 868)
    # Process the input as needed
    result = f'Serial Port Requested for 868 MHz'
    return jsonify(result=result)

@app.route('/delete_serial_868', methods=['GET'])
def delete868():
    # Add your logic here to handle the confirmation
    disconnect_serial("port2")
    result = "Port Disconnected!"
    return jsonify(result=result)

@app.route('/attach_serial_915', methods=['GET'])
def attach_serial_915():
    user_input = escape(request.args.get('user_input'))
    connect_serial(str(user_input), 915)
    # Process the input as needed
    result = f'Serial Port Requested for 915 MHz'
    return jsonify(result=result)

@app.route('/delete_serial_915', methods=['GET'])
def delete915():
    # Add your logic here to handle the confirmation
    disconnect_serial("port3")
    result = "Port Disconnected!"
    return jsonify(result=result)

@socketio.on('connect')
def handle_connect():
    for port, buffer in serial_buffers.items():
        emit(f'initial_serial_data_{port}', {'data': list(buffer)})

@app.route('/transmit433', methods=['POST'])
def transmit433():
    global ser1 
    data = request.json  # Get the data from the POST request
    user_input = data.get('user_input')  # Extract the user input
    msg = "TX:"+user_input+" "
    ser1.write(msg.encode())
    return jsonify(result="Ok")

@app.route('/transmit868', methods=['POST'])
def transmit868():
    global ser2 
    data = request.json  # Get the data from the POST request
    user_input = data.get('user_input')  # Extract the user input
    msg = "TX:"+user_input+" "
    ser2.write(msg.encode())
    return jsonify(result="Ok")

@app.route('/transmit915', methods=['POST'])
def transmit915():
    global ser3
    data = request.json  # Get the data from the POST request
    user_input = data.get('user_input')  # Extract the user input
    msg = "TX:"+user_input+" "
    ser3.write(msg.encode())
    return jsonify(result="Ok")

@app.route('/checkSer', methods=['GET'])
def checkSer():
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
    global surveydata
    print(surveydata)
    return jsonify(surveydata)

if __name__ == '__main__':
    socketio.run(app, debug=True)
