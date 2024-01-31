import requests
from bs4 import BeautifulSoup

# Define the URL and headers
url = "http://10.130.1.1/cgi-bin/log-traffic.has"
headers = {
    "Host": "10.130.1.1",
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:122.0) Gecko/20100101 Firefox/122.0",
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
    "Accept-Language": "en-US,en;q=0.5",
    "Accept-Encoding": "gzip, deflate",
    "DNT": "1",
    "Sec-GPC": "1",
    "Authorization": "Basic cm9vdDpkcmFnaW5v",
    "Connection": "keep-alive",
    "Referer": "http://10.130.1.1/cgi-bin/log-lora.has",
    "Upgrade-Insecure-Requests": "1"
}

# Send the GET request
response = requests.get(url, headers=headers)

if response.status_code == 200:
    # Parse the HTML content using BeautifulSoup
    soup = BeautifulSoup(response.text, 'html.parser')
    
    # Find the table
    table = soup.find('table')
    
    # Initialize an empty list to store formatted strings for each row
    formatted_rows = []
    
    # Find all table rows
    rows = table.find_all('tr')
    
    # Get column headers from the first row
    # Using .text.strip() to clean the text and [1:] to skip the empty first column
    headers = [header.text.strip() for header in rows[0].find_all('th')][1:]
    
    # Iterate through each row (skipping the first row with the headers)
    for row in rows[1:]:
        # Find all data cells (td tags) in the row
        cells = row.find_all('td')
        
        # Extract text from each cell
        # Using [1:] to skip the first cell with the arrow icon
        cell_data = [cell.text.strip() for cell in cells[1:] if cells.index(cell) < len(headers) + 1]
        
        # Format the row data into a neat line
        formatted_row = ' | '.join(cell_data)
        
        # Append the formatted string to the list
        formatted_rows.append(formatted_row)
    
        # Print the formatted string to display the row
        print(formatted_row)
else:
    print("Request failed with status code:", response.status_code)