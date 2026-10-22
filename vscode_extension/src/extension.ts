import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';

export function activate(context: vscode.ExtensionContext) {
    const disposable = vscode.commands.registerCommand('devproxy.openPanel', () => {
        const panel = vscode.window.createWebviewPanel(
            'devproxyPanel',
            'DevProxy Agent Panel',
            vscode.ViewColumn.One,
            {
                enableScripts: true,
                retainContextWhenHidden: true,
            }
        );
        const htmlPath = path.join(context.extensionPath, 'out', 'panelHtml.js');
        // Dynamically import compiled HTML generator (or fallback to raw HTML file)
        const panelHtml = fs.readFileSync(path.join(context.extensionPath, 'panel.html'), 'utf8');
        panel.webview.html = panelHtml;
    });
    context.subscriptions.push(disposable);
}

export function deactivate() {}
