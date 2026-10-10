# DevProxy VS Code Extension

This extension integrates [DevProxy](https://github.com/Aditya-9-6/DevProxy) directly into your VS Code workflow.

## Features

- **Start/Stop DevProxy:** Easily launch or terminate the background proxy engine directly from VS Code.
- **Status Bar Integration:** A convenient status bar item lets you see if the proxy is running and gives you quick access to the control menu.
- **Open Dashboard:** One-click access to open the DevProxy web dashboard in your default browser.

## Requirements

You must have the `devproxy` binary installed on your system.

If it is not in your system's `PATH`, you can configure its location in the extension settings.

## Extension Settings

This extension contributes the following settings:

* `devproxy.binaryPath`: Set the path to the DevProxy executable (default is `devproxy` which assumes it is in your PATH). You can also use paths relative to your workspace like `./devproxy`.
* `devproxy.port`: Port for the HTTP/HTTPS proxy engine (default: `8080`).
* `devproxy.webPort`: Port for the web dashboard and REST/WebSocket API (default: `8081`).

## Commands

- `DevProxy: Start Proxy`
- `DevProxy: Stop Proxy`
- `DevProxy: Open Web Dashboard`

You can also click on the DevProxy item in the status bar at the bottom right to access these commands via a quick menu.
