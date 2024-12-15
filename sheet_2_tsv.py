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


google_sheet_id = config["google_sheet_id"]
worksheet_name = config["google_sheet_name"]

spreadsheet = client.open_by_key(google_sheet_id)
worksheet = spreadsheet.worksheet(worksheet_name)

# Fetch all data
data = worksheet.get_all_values()
headers = data[0]

# Convert rows to a list of dictionaries
row_dicts = [
    {headers[i]: row[i] for i in range(len(headers))}
    for row in data[1:]  # Skip the header row
]

# fetch data which column archived is '' or 0
filtered_data = []
# data's row index
rows_to_update = []
for i, row in enumerate(row_dicts, start=2):  # Skip the header row
    if row.get('archived') in ('', '0'):
        row['archived'] = '1'
        filtered_data.append(row)
        rows_to_update.append(i)

for row in filtered_data:
    print(f"\n{row}")
print(f"\nTotal Rows: {len(filtered_data)}")

# store filtered data to tsv file for import to anki
output_file = './output/cards.tsv'
with open(output_file, 'w', newline='', encoding='utf-8') as file:
    writer = csv.writer(file, delimiter='\t')
    writer.writerow(headers)  # Write headers
    for row in filtered_data:
        writer.writerow(row.values())

# update filtered data column archived as 1
for row_idx in rows_to_update:
    worksheet.update_cell(row_idx, headers.index('archived') + 1, '1')

print(f"Filtered Google Sheets data has been saved as {output_file}")