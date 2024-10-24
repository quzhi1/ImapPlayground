import requests

grant_id = "9bdb7079-0a70-47f0-820a-202a8868f64e"
limit = 10
url = f"https://api.eu.nylas.com/v3/grants/{grant_id}/folders?limit={limit}"
apiKey = "" # put your api key here
payload = {}
headers = {
    'Authorization': 'Bearer ' + apiKey
}

count = 0
url_with_pagination = url
while True:
    response = requests.request("GET", url_with_pagination, headers=headers, data=payload)
    data = response.json()
    object_returned = len(data["data"])
    print(f"Object returned: {object_returned}")
    count += object_returned
    if "next_cursor" in data:
        url_with_pagination = url + '&page_token=' + data["next_cursor"]
    else:
        break

print(f"Total folders: {count}")