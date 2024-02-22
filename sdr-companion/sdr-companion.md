# Introduction

In certain scenarios, traditional LoRa receivers, such as the Adafruit Feather LoRa or Lostik, may not adequately decode LoRa traffic. To address this, we introduce a Software Defined Radio (SDR) companion module designed to offer a more robust alternative for decoding when the conventional LoRa Scanner application falls short.

SDRs provide unparalleled flexibility compared to traditional hardware radios, which are often limited by their specific design and capabilities. With an SDR, you can decode a wide array of signals beyond LoRa, making it an invaluable tool for radio signal research and experimentation. The ability to tweak settings and experiment with different decoding algorithms allows for a deeper exploration into radio communications. This is especially beneficial in the evolving field of LoRa technology, where ongoing research and experimentation can yield significant insights. Projects like the HackRF demonstrate the power of a supportive community, offering extensive resources to enhance your SDR experience.

## gr-lora

The **gr-lora** project comprises GNU Radio blocks designed to decode LoRa-modulated radio messages using an SDR. LoRa (Long Range) is crucial for Internet of Things (IoT) applications due to its long-range, low-power, and low-bitrate communication capabilities.

While gr-lora supports the majority of LoRa's physical-layer modulation features, it does not facilitate CRC checks of payload and header or decode multiple channels simultaneously. Tested primarily with a USRP B201 and a Microchip RN2483 transmitter, it is also compatible with popular SDRs like the HackRF.

A standout feature of gr-lora is its clock recovery algorithm, introduced in version 0.6, enhancing the handling of long LoRa messages. Improvements in whitening, detection, and decoding further solidify its utility in IoT applications. This ongoing project exemplifies the collaborative spirit of the open-source community, continually evolving to meet users' needs.

## Running the sdr-companion

Ensure [Docker](https://www.docker.com/get-started/) is installed on your system to proceed with the sdr-companion setup.

1. **Download the Docker Image**: Begin by downloading the gr-lora image using the following command:

   ```bash
   docker pull rpp0/gr-lora:latest
   ```

2. **Prepare Your SDR**: Connect your SDR (e.g., HackRF) to your machine. Ensure device permissions are correctly set to avoid detection issues.

3. **Run the Docker Container**: Execute the script to start the Docker container with gr-lora:

   ```bash
   ./docker_run_grlora.sh
   ```

   Upon success, you'll enter the Docker container environment:

   ```bash
   [root@9089ddfe32b5 apps]#
   ```

4. **Launch GNU Radio Companion**: Inside the container, open the lora_receive_realtime.grc flowgraph with GNU Radio Companion:

   ```bash
   gnuradio-companion lora_receive_realtime.grc
   ```

5. **Configure and Execute**: Adjust the frequency, spreading factor, and other settings as needed at the gnuradio-companion interface. Start receiving and decoding by selecting *Run -> Execute*.

   Decoded traffic will be displayed in hex format in the gnuradio-companion console.

## Prebuilt Flowgraphs

For convenience, prebuilt flowgraphs for specific frequencies are available:

- **915 MHz SF7 125k BW CR 4/5**: `915-sf7-125bw-cr4_5.grc`
- **868 MHz SF7 125k BW CR 4/5**: `868-sf7-125bw-cr4_5.grc`

To download these into the Docker container:

1. **Identify Host IP**: Use `ip addr` (or `ifconfig` if available) to find your host machine's IP address.

2. **Host Web Server**: On your host machine, in the directory with the flowgraphs, start a web server:

   ```bash
   python3 -m http.server 80
   ```

3. **Download Flowgraphs**: Inside the container, use curl to download the desired flowgraph:

   ```bash
   curl -O http://<your-host-ip>/915-sf7-125bw-cr4_5.grc
   curl -O http://<your-host-ip>/868-sf7-125bw-cr4_5.grc
   ```

## Troubleshooting and Support

For troubleshooting common issues or seeking additional guidance, refer to the gr-lora project documentation and community forums. These resources can provide valuable support as you navigate installation and operation challenges.
