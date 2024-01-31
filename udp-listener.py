import socket

# Define the IP address and the port number to listen on.
# '' means the script will listen on all available IPs.
IP = '10.130.1.235'
PORT = 1700

def main():
    # Create a UDP socket
    with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as s:
        # Bind the socket to the address and port
        s.bind((IP, PORT))
        print(f"Listening for UDP traffic on port {PORT}...")

        # Continuously listen for UDP packets
        while True:
            # Receive data from the client
            data, addr = s.recvfrom(1024)  # buffer size is 1024 bytes
            print(f"Received message from {addr}: {data}")

if __name__ == "__main__":
    main()
