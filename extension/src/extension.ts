// @ts-ignore
import * as vscode from 'vscode';
// @ts-ignore
import * as path from 'path';
// @ts-ignore
import * as fs from 'fs';
import { CLIWrapper, HardFaultResult } from './modules/cli_wrapper';
import { FaultPanel } from './panels/fault_panel';
import { BootstrapService } from './modules/bootstrap';

export async function activate(context: vscode.ExtensionContext) {
    console.log('HARDCOREAI extension is now active');

    // Bootstrap Intelligence Layer (Async)
    const pythonReady = BootstrapService.ensureReady(context);

    // Decoration for crash line
    const crashDecorationType = vscode.window.createTextEditorDecorationType({
        backgroundColor: 'rgba(255, 0, 0, 0.2)',
        isWholeLine: true,
        overviewRulerColor: 'red',
        overviewRulerLane: vscode.OverviewRulerLane.Full,
        after: {
            contentText: ' ← HARDCOREAI: CRASH SITE',
            color: 'rgba(255, 0, 0, 0.8)',
            margin: '0 0 0 1em'
        }
    });

    /**
     * Extracts a code snippet around a specific line in a file.
     */
    function getCodeSnippet(filePath: string, line: number, contextLines: number = 3): string {
        try {
            if (!fs.existsSync(filePath)) return "";
            const content = fs.readFileSync(filePath, 'utf8');
            const lines = content.split(/\r?\n/);
            
            const start = Math.max(0, line - contextLines - 1);
            const end = Math.min(lines.length, line + contextLines);
            
            let snippet = "";
            for (let i = start; i < end; i++) {
                const prefix = (i === line - 1) ? "→" : " ";
                snippet += `${prefix} ${i + 1}: ${lines[i]}\n`;
            }
            return snippet;
        } catch (e) {
            return `[Error reading code snippet: ${e}]`;
        }
    }

    /**
     * Opens a file and highlights a specific line.
     */
    async function openSource(filePath: string, line: number) {
        if (!fs.existsSync(filePath)) {
            vscode.window.showErrorMessage(`Source file not found: ${filePath}`);
            return;
        }

        const uri = vscode.Uri.file(filePath);
        const doc = await vscode.workspace.openTextDocument(uri);
        const editor = await vscode.window.showTextDocument(doc);
        
        const position = new vscode.Position(line - 1, 0);
        editor.selection = new vscode.Selection(position, position);
        editor.revealRange(new vscode.Range(position, position), vscode.TextEditorRevealType.InCenter);

        // Apply decoration
        editor.setDecorations(crashDecorationType, [new vscode.Range(position, position)]);
    }

    /**
     * Core analysis logic shared by commands.
     */
    async function runAnalysis(faultJson: string, elfPath?: string) {
        await vscode.window.withProgress({
            location: vscode.ProgressLocation.Notification,
            title: "HARDCOREAI: Performing Deep Analysis...",
            cancellable: false
        }, async (progress) => {
            try {
                let result: HardFaultResult;

                // SMART DETECTION: If JSON already has 'fault_type', it's a finished report
                try {
                    const parsed = JSON.parse(faultJson);
                    if (parsed.fault_type) {
                        result = parsed as HardFaultResult;
                    } else {
                        result = await CLIWrapper.decodeHardFault(faultJson, elfPath);
                    }
                } catch (e) {
                    result = await CLIWrapper.decodeHardFault(faultJson, elfPath);
                }
                // Enhance result with code snippet if where.file exists
                if (result.where.file && result.where.line) {
                    // Try to find the file in workspace
                    const files = await vscode.workspace.findFiles(`**/${result.where.file}`);
                    if (files.length > 0) {
                        const fullPath = files[0].fsPath;
                        result.code_snippet = getCodeSnippet(fullPath, result.where.line);
                        // Store the full path for opening later
                        (result.where as any).fullPath = fullPath;
                    }
                }

                FaultPanel.createOrShow(result);
                
                // Setup listener for source opening
                FaultPanel.onOpenSource((data) => {
                    if (data.fullPath) {
                        openSource(data.fullPath, data.line);
                    } else if (data.file && data.line) {
                        // Fallback search if fullPath not provided
                        vscode.workspace.findFiles(`**/${data.file}`).then(uris => {
                            if (uris.length > 0) {
                                openSource(uris[0].fsPath, data.line);
                            }
                        });
                    }
                });

            } catch (e: any) {
                vscode.window.showErrorMessage(`Analysis Failed: ${e.message}`);
            }
        });
    }

    // Command: Analyze Current Project (One-Click)
    let analyzeProjectCommand = vscode.commands.registerCommand('hardcoreai.analyzeProject', async () => {
        try {
            const faultFiles = await vscode.workspace.findFiles('**/fault*.json', '**/node_modules/**');
            if (faultFiles.length === 0) {
                vscode.window.showErrorMessage("No fault JSON found in workspace (expected fault*.json)");
                return;
            }

            const faultFile = faultFiles[0];
            const faultData = fs.readFileSync(faultFile.fsPath, 'utf8');

            const elfFiles = await vscode.workspace.findFiles('**/*.elf', '**/node_modules/**');
            let elfPath = elfFiles.length > 0 ? elfFiles[0].fsPath : undefined;

            await runAnalysis(faultData, elfPath);
        } catch (e: any) {
            vscode.window.showErrorMessage(`Project Analysis Error: ${e.message}`);
        }
    });

    let decodeCommand = vscode.commands.registerCommand('hardcoreai.decodeHardFault', async () => {
        const workspaceFolders = vscode.workspace.workspaceFolders;
        if (!workspaceFolders) return;

        const detectedFiles = [
            ...await vscode.workspace.findFiles('**/*fault*.json', '**/node_modules/**'),
            ...await vscode.workspace.findFiles('**/*dump*.json', '**/node_modules/**')
        ];

        const items = [{ label: '$(edit) Paste JSON' }, { label: '$(file-directory) Browse Workspace...' }];
        detectedFiles.forEach(f => items.push({ 
            label: `$(file) ${path.basename(f.fsPath)}`, 
            description: vscode.workspace.asRelativePath(f) 
        } as any));

        const selection = await vscode.window.showQuickPick(items as any);
        if (!selection) return;

        let faultData: string | undefined;
        if ((selection as any).label.includes('Paste')) {
            faultData = await vscode.window.showInputBox({ prompt: 'Paste JSON' });
        } else if ((selection as any).description) {
            const fullPath = path.join(workspaceFolders[0].uri.fsPath, (selection as any).description);
            faultData = fs.readFileSync(fullPath, 'utf8');
        } else {
             const uris = await vscode.window.showOpenDialog({ filters: { 'JSON': ['json'] } });
             if (uris) faultData = fs.readFileSync(uris[0].fsPath, 'utf8');
        }
        
        if (!faultData) return;

        const elfFiles = await vscode.workspace.findFiles('**/*.elf');
        const elfItems = elfFiles.map(f => ({ label: path.basename(f.fsPath), path: f.fsPath }));
        const elfSelect = await vscode.window.showQuickPick([{ label: 'Skip ELF' }, ...elfItems] as any);
        
        await runAnalysis(faultData, (elfSelect as any)?.path);
    });

    // Command: Analyze a specific file (used by auto-detect)
    let analyzeFileCommand = vscode.commands.registerCommand('hardcoreai.analyzeFile', async (fileUri: vscode.Uri) => {
        try {
            const faultData = fs.readFileSync(fileUri.fsPath, 'utf8');
            const elfFiles = await vscode.workspace.findFiles('**/*.elf');
            const elfPath = elfFiles.length > 0 ? elfFiles[0].fsPath : undefined;
            await runAnalysis(faultData, elfPath);
        } catch (e: any) {
            vscode.window.showErrorMessage(`File Analysis Error: ${e.message}`);
        }
    });

    // Auto-Crash Detection: Watch for auto_fault.json
    const faultWatcher = vscode.workspace.createFileSystemWatcher('**/auto_fault.json');
    faultWatcher.onDidCreate(uri => {
        vscode.window.showInformationMessage("💥 HARDWARE CRASH DETECTED! Analyzing now...", "View Report").then(selection => {
            if (selection === "View Report") {
                vscode.commands.executeCommand('hardcoreai.analyzeFile', uri);
            }
        });
    });
    // Also trigger on change (if file is reused)
    faultWatcher.onDidChange(uri => vscode.commands.executeCommand('hardcoreai.analyzeFile', uri));

    let liveMonitorCommand = vscode.commands.registerCommand('hardcoreai.startLiveMonitor', async () => {
        const interfaceCfg = await vscode.window.showInputBox({ 
            value: 'interface/stlink.cfg', 
            prompt: 'OpenOCD Interface (e.g. interface/stlink.cfg)' 
        });
        const targetCfg = await vscode.window.showInputBox({ 
            value: 'target/stm32f1x.cfg', 
            prompt: 'OpenOCD Target (e.g. target/stm32f1x.cfg)' 
        });
        if (!interfaceCfg || !targetCfg) return;

        const elfFiles = await vscode.workspace.findFiles('**/*.elf');
        const elfPath = elfFiles.length > 0 ? elfFiles[0].fsPath : undefined;

        const config = vscode.workspace.getConfiguration('hardcoreai');
        const apiKey = config.get<string>('geminiApiKey');

        // Terminal-based monitor for visibility (FORCE WORKSPACE CWD)
        const workspacePath = vscode.workspace.workspaceFolders ? vscode.workspace.workspaceFolders[0].uri.fsPath : undefined;
        const terminal = vscode.window.createTerminal({
            name: "HARDCOREAI: Live Monitor",
            cwd: workspacePath,
            env: {
                GEMINI_API_KEY: apiKey || ""
            }
        });
        terminal.show();
        
        // Find internal backend path
        const pythonPath = await pythonReady; // Wait for bootstrap
        const cliPath = path.join(context.extensionPath, "backend", "cli", "hc_fault.py");
        
        const isWindows = process.platform === 'win32';
        const callOp = isWindows ? '& ' : '';
        let cmd = `${callOp}"${pythonPath}" "${cliPath}" debug live --interface ${interfaceCfg} --target ${targetCfg}`;
        if (elfPath) cmd += ` --elf "${elfPath}"`;
        
        terminal.sendText(cmd);
        vscode.window.showInformationMessage("HARDCOREAI: Live hardware monitoring started.");
    });

    context.subscriptions.push(
        analyzeProjectCommand, 
        decodeCommand, 
        liveMonitorCommand, 
        analyzeFileCommand,
        faultWatcher
    );

    // Auto-detection on Startup (fault.json)
    vscode.workspace.findFiles('**/fault*.json').then(files => {
        if (files.length > 0) {
            vscode.window.showInformationMessage(`HARDCOREAI: Detected fault dump (${path.basename(files[0].fsPath)}).`, "Analyze Now").then(choice => {
                if (choice === "Analyze Now") {
                    vscode.commands.executeCommand('hardcoreai.analyzeProject');
                }
            });
        }
    });
}

export function deactivate() {}
