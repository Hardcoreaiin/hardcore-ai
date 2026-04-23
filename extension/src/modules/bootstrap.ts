import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as path from 'path';
import * as fs from 'fs';
import { promisify } from 'util';

const exec = promisify(cp.exec);

export class BootstrapService {
    public static async ensureReady(context: vscode.ExtensionContext): Promise<string> {
        const backendDir = path.join(context.extensionPath, 'backend');
        const venvDir = path.join(backendDir, '.venv');
        const pythonInVenv = process.platform === 'win32' 
            ? path.join(venvDir, 'Scripts', 'python.exe') 
            : path.join(venvDir, 'bin', 'python');

        if (fs.existsSync(pythonInVenv)) {
            return pythonInVenv;
        }

        return await vscode.window.withProgress({
            location: vscode.ProgressLocation.Notification,
            title: "Setting up HARDCOREAI Intelligence Layer...",
            cancellable: false
        }, async (progress) => {
            try {
                // 1. Detect System Python
                progress.report({ message: "Detecting System Python..." });
                const systemPython = await this.getSystemPython();
                
                // 2. Create Venv
                progress.report({ message: "Creating isolated environment..." });
                await exec(`"${systemPython}" -m venv "${venvDir}"`);
                
                // 3. Install Requirements
                progress.report({ message: "Installing core diagnostics (30s)..." });
                const pipPath = process.platform === 'win32'
                    ? path.join(venvDir, 'Scripts', 'pip.exe')
                    : path.join(venvDir, 'bin', 'pip');
                
                const reqPath = path.join(backendDir, 'requirements.txt');
                await exec(`"${pipPath}" install -r "${reqPath}"`);
                
                vscode.window.showInformationMessage("HARDCOREAI Intelligence Layer Ready.");
                return pythonInVenv;
            } catch (error: any) {
                vscode.window.showErrorMessage(`Failed to bootstrap HARDCOREAI: ${error.message}`);
                throw error;
            }
        });
    }

    private static async getSystemPython(): Promise<string> {
        const commands = process.platform === 'win32' ? ['python', 'py -3', 'python3'] : ['python3', 'python'];
        for (const cmd of commands) {
            try {
                await exec(`${cmd} --version`);
                return cmd;
            } catch {}
        }
        throw new Error("Python 3 not found on your system. Please install it from python.org.");
    }
}
