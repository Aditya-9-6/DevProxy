import * as assert from 'assert';

// You can import and use all API from the 'vscode' module
// as well as import your extension to test it
import * as vscode from 'vscode';

suite('Extension Test Suite', () => {
    vscode.window.showInformationMessage('Start all tests.');

    test('Extension should be present', () => {
        assert.ok(vscode.extensions.getExtension('aditya-9-6.devproxy'));
    });

    test('Should register commands', async () => {
        const commands = await vscode.commands.getCommands(true);
        const devproxyCommands = commands.filter(c => c.startsWith('devproxy.'));

        assert.ok(devproxyCommands.includes('devproxy.start'));
        assert.ok(devproxyCommands.includes('devproxy.stop'));
        assert.ok(devproxyCommands.includes('devproxy.openDashboard'));
    });
});
