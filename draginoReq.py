import requests
from bs4 import BeautifulSoup

def get_gateway_data(url, headers):
    """
    Makes a request to the specified gateway URL and parses the HTML table content.
    Returns a list of formatted strings for each row in the table.
    """
    formatted_rows = []
    response = requests.get(url, headers=headers)

    if response.status_code == 200:
        soup = BeautifulSoup(response.text, 'html.parser')
        table = soup.find('table')
        if table:
            rows = table.find_all('tr')
            headers = [header.text.strip() for header in rows[0].find_all('th')][1:]
            for row in rows[1:]:
                cells = row.find_all('td')
                cell_data = [cell.text.strip() for cell in cells[1:] if cells.index(cell) < len(headers) + 1]
                formatted_row = ' | '.join(cell_data)
                formatted_rows.append(formatted_row)
    else:
        print(f"Request to {url} failed with status code:", response.status_code)
    
    return formatted_rows


# Define URLs and headers for both gateways
gateway_1 = "http://192.168.1.23/cgi-bin/log-traffic.has"
gateway_2 = "http://192.168.1.24/cgi-bin/log-traffic.has"
headers = {
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:122.0) Gecko/20100101 Firefox/122.0",
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
    "Accept-Language": "en-US,en;q=0.5",
    "Accept-Encoding": "gzip, deflate",
    "DNT": "1",
    "Sec-GPC": "1",
    "Authorization": "Basic cm9vdDpkcmFnaW5v",
    "Connection": "keep-alive",
    "Upgrade-Insecure-Requests": "1"
}

# Fetch and print data from both gateways
print("Fetching data from Gateway 1 (192.168.1.23)...")
data_1 = get_gateway_data(gateway_1, headers)
for row in data_1:
    print(row)

print("\nFetching data from Gateway 2 (192.168.1.24)...")
# Update the 'Host' header for the second gateway if necessary
headers["Host"] = "192.168.1.24"
data_2 = get_gateway_data(gateway_2, headers)
for row in data_2:
    print(row)


