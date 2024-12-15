import csv
import json
import gspread
from oauth2client.service_account import ServiceAccountCredentials

scope = [
    "https://www.googleapis.com/auth/spreadsheets",
    "https://www.googleapis.com/auth/drive"
]

creds = ServiceAccountCredentials.from_json_keyfile_name(
    "./config/anki-en-credential.json", scope
)
client = gspread.authorize(creds)


with open("./config/config.json", "r") as config_file:
    config = json.load(config_file)

# Read Google Sheet ID and worksheet name from the config
google_sheet_id = config["google_sheet_id"]
worksheet_name = config["anki-en"]

# Open the Google Sheet and select the worksheet
spreadsheet = client.open_by_key(google_sheet_id)
worksheet = spreadsheet.worksheet(worksheet_name)

# Fetch all data from the worksheet
data = worksheet.get_all_values()

headers = data[0]

# Convert rows to a list of dictionaries
row_dicts = [
    {headers[i]: row[i] for i in range(len(headers))}
    for row in data[1:]  # Skip the header row
]

# Filter rows where 'archived' column is empty or '0'
filtered_data = [
    row for row in row_dicts
    if row.get('archived') in ('', '0')
]


for row in filtered_data:
    print(f"\n{row}")
print(f"\nTotal Rows: {len(filtered_data)}")


output_file = 'output.tsv'
with open(output_file, 'w', newline='', encoding='utf-8') as file:
    writer = csv.writer(file, delimiter='\t')
    writer.writerow(headers)  # Write headers
    for row in filtered_data:
        writer.writerow(row.values())

print(f"Filtered Google Sheets data has been saved as {output_file}")