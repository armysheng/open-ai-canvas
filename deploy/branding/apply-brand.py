"""Apply the brand package through the existing admin API; never print credentials."""
import argparse
import datetime
import http.cookiejar
import json
from pathlib import Path
import urllib.request
import uuid


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", required=True)
    parser.add_argument("--credentials-file", required=True, type=Path)
    parser.add_argument("--backup-dir", required=True, type=Path)
    args = parser.parse_args()
    base = args.base_url.rstrip("/")
    if not base.startswith("https://"):
        parser.error("The deployed site must use HTTPS")
    package = Path(__file__).resolve().parent
    brand = json.loads((package / "brand.json").read_text())
    credentials = json.loads(args.credentials_file.read_text())
    jar = http.cookiejar.CookieJar()
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))

    def request(route, payload=None, method=None, content_type="application/json"):
        body = payload if isinstance(payload, bytes) else None if payload is None else json.dumps(payload).encode()
        req = urllib.request.Request(base + "/api" + route, data=body, method=method,
                                     headers={"Content-Type": content_type, "Origin": base})
        with opener.open(req, timeout=30) as response:
            result = json.load(response)
        if result.get("code") != 0:
            raise RuntimeError(f"API rejected {route}: code {result.get('code')}")
        return result["data"]

    user = request("/auth/login", credentials)["user"]
    try:
        if user["role"] != "admin":
            raise RuntimeError("An administrator account is required")
        previous = request("/admin/settings/appearance")["setting"]
        args.backup_dir.mkdir(parents=True, exist_ok=True, mode=0o700)
        stamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        backup = args.backup_dir / f"appearance-before-{stamp}.json"
        with backup.open("x") as output:
            backup.chmod(0o600)
            json.dump(previous, output, ensure_ascii=False, indent=2)

        for slot, name, field in [("logo", "logo-light.png", "logoResourceId"),
                                  ("logo-dark", "logo-dark.png", "darkLogoResourceId")]:
            boundary = "brand-" + uuid.uuid4().hex
            header = (f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; "
                      f"filename=\"{name}\"\r\nContent-Type: image/png\r\n\r\n").encode()
            payload = header + (package / name).read_bytes() + f"\r\n--{boundary}--\r\n".encode()
            resource = request(f"/admin/settings/appearance/assets/{slot}", payload,
                               content_type=f"multipart/form-data; boundary={boundary}")["resource"]
            brand[field] = resource["id"]

        setting = request("/admin/settings/appearance", brand, "PATCH")["setting"]
        for field in ("skinId", "skinThemes", "authHeroTitle", "authHeroDescription",
                      "authVideoResourceId", "authVideoPosterResourceId", "authVideoAutoplay"):
            if setting[field] != previous[field]:
                raise RuntimeError(f"Unexpected change to {field}; backup: {backup}")
        public = request("/public/appearance")["appearance"]
        assert public["brandName"] == brand["brandName"]
        assert public["logoConfigured"] and public["darkLogoConfigured"]
        print(json.dumps({"brandName": public["brandName"], "skinId": public["skinId"],
                          "logosConfigured": True, "backup": str(backup)}, ensure_ascii=False))
    finally:
        request("/auth/logout", {})


if __name__ == "__main__":
    main()
