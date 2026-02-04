import requests


url = "https://api.sorai.me/get-sora-link"
payload = {
    "url": "https://sora.chatgpt.com/p/s_6981a0e0c60c8191826fa65ef36ed09f",
    "token": "sk-jS-eKjfjOlvcsGJRZxeTLIRUgDZ57n4ZF06omdACqNg",
}

resp = requests.post(url, json=payload, timeout=30)
print("status:", resp.status_code)
print("body:", resp.text)