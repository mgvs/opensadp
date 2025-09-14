# OpenSADP (Go)

OpenSADP is a Go reimplementation, designed to run sort of SADP Tool by Hikvision at MacBook (aarch64).

- Core: SADP UDP multicast scan and XML parsing in pure Go
- UI: Using miqt (Qt bindings for Go) — see `github.com/mappu/miqt`

## Status

- Core SADP scan implemented in `sadp/`
- CLI scanner `cmd/opensadp-scan` prints discovered devices as CSV
- Qt UI scaffolding will follow using miqt: `https://github.com/mappu/miqt`

## Build

Go 1.22+ recommended.

```bash
cd opensadp
go build -ldflags "-s -w" -o OpenSADP ./cmd/opensadp-qt && go build -ldflags "-s -w" -o opensadp-scan ./cmd/opensadp-scan
./opensadp-scan
```

## Resources (icon) embedding

OpenSADP embeds the application icon into the binary using miqt resources, so the `.ico` file is not required next to the executable.

- Icon source: `res/sadp.ico`
- Resource manifest: `res/resources.qrc`
- Generated Go file: `res/resources_gen.go`
- Resource path used in code: `:/opensadp/sadp.ico`

If you update files under `res/`, regenerate the embedded resource:

```bash
cd opensadp
# Generate Go resources using miqt-rcc
go run github.com/mappu/miqt/cmd/miqt-rcc@latest \
  -Input res/resources.qrc \
  -OutputGo res/resources_gen.go \
  -Package res

# Rebuild the UI app
go build -ldflags "-s -w" -o OpenSADP ./cmd/opensadp-qt
```

At runtime the app loads the icon from the embedded Qt resource path `:/opensadp/sadp.ico`.

## Trademark / ownership notice

The **SADP Tool** (software used for searching the online devices in the same network. It supports viewing the device information, activating the device, editing the network parameters of the device and resetting the device password, etc.) is a product of **Hikvision** and belongs to Hikvision.  
The name **OpenSADP** was chosen to give users an idea of the program's functionality and is not intended to imply any endorsement by or affiliation with Hikvision.

---

## How third-party licenses are included

If OpenSADP includes third-party libraries, their license texts and copyright notices must be
preserved. Below are example entries — replace them with the actual libraries used in your build.

---

## Notes

- On some systems you may need firewall permissions to receive UDP replies.
- For Qt UI with miqt on macOS (dynamic): install Homebrew `golang`, `pkg-config`, and `qt@5`, then follow miqt README build guidance.

References:
- miqt (Qt bindings for Go): `https://github.com/mappu/miqt`
