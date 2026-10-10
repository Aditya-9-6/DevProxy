import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as path from 'path';

let proxyProcess: cp.ChildProcess | null = null;
let statusBarItem: vscode.StatusBarItem;

export function activate(context: vscode.ExtensionContext) {
    // Create Status Bar Item
    statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
    statusBarItem.command = 'devproxy.showMenu';
    context.subscriptions.push(statusBarItem);
    updateStatusBar(false);

    // Register Commands
    const startCmd = vscode.commands.registerCommand('devproxy.start', startProxy);
    const stopCmd = vscode.commands.registerCommand('devproxy.stop', stopProxy);
    const dashboardCmd = vscode.commands.registerCommand('devproxy.openDashboard', openDashboard);

    const menuCmd = vscode.commands.registerCommand('devproxy.showMenu', async () => {
        const isRunning = proxyProcess !== null;

        const options: vscode.QuickPickItem[] = [];

        if (isRunning) {
            options.push({ label: '$(browser) Open Dashboard', description: 'Open DevProxy Web UI in browser' });
            options.push({ label: '$(stop-circle) Stop Proxy', description: 'Stop the DevProxy engine' });
        } else {
            options.push({ label: '$(play-circle) Start Proxy', description: 'Start the DevProxy engine' });
        }

        const selection = await vscode.window.showQuickPick(options, { placeHolder: 'DevProxy Controls' });

        if (selection) {
            if (selection.label.includes('Start')) {
                startProxy();
            } else if (selection.label.includes('Stop')) {
                stopProxy();
            } else if (selection.label.includes('Dashboard')) {
                openDashboard();
            }
        }
    });

    context.subscriptions.push(startCmd, stopCmd, dashboardCmd, menuCmd);
}

function updateStatusBar(isRunning: boolean) {
    if (isRunning) {
        statusBarItem.text = '$(shield) DevProxy: Running';
        statusBarItem.tooltip = 'Click to open DevProxy menu';
        statusBarItem.backgroundColor = undefined;
    } else {
        statusBarItem.text = '$(shield) DevProxy: Stopped';
        statusBarItem.tooltip = 'Click to open DevProxy menu';
        statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground');
    }
    statusBarItem.show();
}

function startProxy() {
    if (proxyProcess) {
        vscode.window.showInformationMessage('DevProxy is already running.');
        return;
    }

    const config = vscode.workspace.getConfiguration('devproxy');
    let binaryPath = config.get<string>('binaryPath') || 'devproxy';

    // Resolve relative paths based on workspace root if it starts with .
    if (binaryPath.startsWith('.') && vscode.workspace.workspaceFolders && vscode.workspace.workspaceFolders.length > 0) {
        binaryPath = path.resolve(vscode.workspace.workspaceFolders[0].uri.fsPath, binaryPath);
    }

    const port = config.get<number>('port') || 8080;
    const webPort = config.get<number>('webPort') || 8081;

    const args = [
        '-port', port.toString(),
        '-web-port', webPort.toString()
    ];

    try {
        const cwd = vscode.workspace.workspaceFolders ? vscode.workspace.workspaceFolders[0].uri.fsPath : process.cwd();

        proxyProcess = cp.spawn(binaryPath, args, { cwd });

        proxyProcess.on('error', (err) => {
            vscode.window.showErrorMessage(`Failed to start DevProxy: ${err.message}. Ensure '${binaryPath}' is installed and in your PATH, or update the binaryPath setting.`);
            proxyProcess = null;
            updateStatusBar(false);
        });

        proxyProcess.on('exit', (code, signal) => {
            const msg = code !== null ? `with code ${code}` : `by signal ${signal}`;
            if (code !== 0 && code !== null) {
                vscode.window.showWarningMessage(`DevProxy exited ${msg}`);
            } else {
                vscode.window.showInformationMessage(`DevProxy stopped ${msg}`);
            }
            proxyProcess = null;
            updateStatusBar(false);
        });

        // Basic check if it started
        if (proxyProcess.pid) {
            vscode.window.showInformationMessage(`DevProxy started on port ${port} (Web UI: ${webPort})`);
            updateStatusBar(true);

            // Optionally auto-open dashboard or ask
            vscode.window.showInformationMessage('DevProxy is running. Open Dashboard?', 'Yes', 'No').then(selection => {
                if (selection === 'Yes') {
                    openDashboard();
                }
            });
        }
    } catch (e) {
        vscode.window.showErrorMessage(`Error launching DevProxy: ${e}`);
        proxyProcess = null;
        updateStatusBar(false);
    }
}

function stopProxy() {
    if (!proxyProcess) {
        vscode.window.showInformationMessage('DevProxy is not running.');
        return;
    }

    proxyProcess.kill('SIGTERM');
    // The exit event handler will clean up state
}

function openDashboard() {
    const config = vscode.workspace.getConfiguration('devproxy');
    const webPort = config.get<number>('webPort') || 8081;
    vscode.env.openExternal(vscode.Uri.parse(`http://localhost:${webPort}`));
}

export function deactivate() {
    if (proxyProcess) {
        proxyProcess.kill('SIGTERM');
        proxyProcess = null;
    }
}
