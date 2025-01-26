import os
import requests
import json

def send_file_to_lambda(file_path, lambda_url):
    try:
        # Read the JSON file content
        with open(file_path, 'r') as f:
            file_content = f.read()

        # Make a POST request to the Lambda function
        response = requests.post(lambda_url, json={"body": file_content})

        # Check the response status
        if response.status_code == 200:
            print(f"Successfully uploaded {os.path.basename(file_path)}")
            return True
        else:
            print(f"Failed to upload {os.path.basename(file_path)}: {response.text}")
            return False
    except Exception as e:
        print(f"Error processing {file_path}: {str(e)}")
        return False

def main():
    matches_directory = "matches"
    lambda_url = "https://v5gegme2a6.execute-api.eu-north-1.amazonaws.com/default/sendToBucket"

    # File IDs (without .json extension)
    file_ids = [
        "4WaP4sDDTbGG32NSefYZCTTLAD7oapftaBs3uipWt3SfNjtrDbAY6Csq2Kubw1pAnKR8GjB6AY2GWSj4BtR3RWdq",
        "5WsaAedparyv3vJMqXNGFJSgaP2qFeJTsssYZodjXK4juhgUu6rAAWBQot3wP369WkATqSunaxDevZVqYYLnnkm6",
        "Y5bZFbrQmjKpCcE6CUGNHSAZgFj2yQmQoVWJk3xYQdtXDKtw2noE5kMHM8EDR4oyCUSCefknhWxbxehLjzP65Cu",
        "5G3Y8jcfG8myzCAdkuy5iRxdXnf2fZQWX2268rS1hYr6CMvpt32fUmzQBi9QwUrdwzJZ2LrBoAfszr562a7sAyAH"
    ]

    if not os.path.exists(matches_directory):
        print(f"Directory '{matches_directory}' does not exist.")
        return

    for file_id in file_ids:
        file_name = file_id + ".json"
        file_path = os.path.join(matches_directory, file_name)

        if os.path.isfile(file_path):
            # Send the file to the Lambda function
            if send_file_to_lambda(file_path, lambda_url):
                # If successful, delete the file
                os.remove(file_path)
        else:
            print(f"File not found: {file_name}")

    # Check if any files remain in the directory
    remaining_files = os.listdir(matches_directory)
    if not remaining_files:
        print("All specified files were processed and the directory is now empty.")
    else:
        print(f"Some files remain in the directory: {remaining_files}")

if __name__ == "__main__":
    main()
