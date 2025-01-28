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
        "2FjoEdzowSoJEiAoUf1NqAp3ZS2jXrZPy3kt85ZWrRyasC6SVKuen1nRLvM4AWLbWYkw9CJhKZa9RLZpsQV3zY7E", "sQcu476aUzLSYyRYC4Zo5zmdXibEHhVyP4U5onUdbtFHqb12fhqNg1rG241CFtaVFpsXax8TFJ3wc1yB2feqtmN", "RkwYUmP4FvWjJB9AA6fyXgbRpmpcrBm1WpuQnG96cwJxxPVpHMouAGJz3QbT8YoMSR4g5Y3pDq856ZVc8dvB5kY"
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

if __name__ == "__main__":
    main()
