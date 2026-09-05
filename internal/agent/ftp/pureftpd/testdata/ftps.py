"""FTPS wire checks for the opt-in, disposable Ubuntu integration test."""

import ftplib
import io
import json
import ssl
import sys

config = json.loads(sys.stdin.readline())
mode = config["mode"]
context = ssl.create_default_context(cafile=config["certificate"])
context.minimum_version = ssl.TLSVersion.TLSv1_2
client = ftplib.FTP() if mode == "plaintext" else ftplib.FTP_TLS(context=context)
# Pure-FTPd intentionally delays failed logins as a brute-force mitigation.
client.connect("127.0.0.1", config["port"], timeout=15)

if mode in ("plaintext", "denied"):
    try:
        client.login(config["username"], config["password"])
    except (ftplib.error_perm, ftplib.error_temp) as error:
        assert str(error).startswith("530") or (
            mode == "plaintext" and str(error).startswith("421")
        ), "unexpected authentication failure"
        print("denied", flush=True)
    else:
        raise AssertionError("unauthorized login succeeded")
    finally:
        client.close()
    sys.exit(0)

client.login(config["username"], config["password"])
client.prot_p()

if mode in ("revoked", "retained"):
    print("ready", flush=True)
    assert sys.stdin.readline().strip() == "check"
    try:
        client.voidcmd("NOOP")
    except (EOFError, OSError, ftplib.error_temp):
        assert mode == "revoked", "another account's session was terminated"
    else:
        assert mode == "retained", "revoked FTP session remained connected"
    client.close()
    sys.exit(0)

payload = b"WEBYCP encrypted transfer\n"
client.storbinary("STOR upload.txt", io.BytesIO(payload))
received = io.BytesIO()
client.retrbinary("RETR upload.txt", received.write)
assert received.getvalue() == payload, "transfer contents changed"
assert client.pwd() == "/", "FTP login was not chrooted"
client.cwd("../../../../")
assert client.pwd() == "/", "FTP traversal escaped the jail"
for path in ("/etc/passwd", "escape/passwd"):
    try:
        client.retrbinary("RETR " + path, lambda _: None)
    except ftplib.error_perm:
        pass
    else:
        raise AssertionError("FTP read outside the account jail")

client.prot_c()
try:
    client.nlst()
except ftplib.error_perm:
    pass
else:
    raise AssertionError("unencrypted data transfer succeeded")
client.close()
print("transfer verified", flush=True)
