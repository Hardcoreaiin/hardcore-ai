// @ts-ignore
import * as cp from 'child_process';
// @ts-ignore
import * as path from 'path';
// @ts-ignore
import * as vscode from 'vscode';
// @ts-ignore
import * as fs from 'fs';
// @ts-ignore
import * as os from 'os';

export interface HardFaultResult {
    fault_type: string;
    what_happened: string;
    likely_scenario: string;
    memory_analysis: {
        region: string;
        description: string;
        valid: string;
        executable: string;
    };
    where: {
        function: string;
        address: string;
        file?: string;
        line?: number;
    };
    fix: string[];
    confidence: "Deterministic" | "Heuristic";
    raw_details: Array<{
        name: string;
        description: string;
        register: string;
    }>;
    raw_registers: any;
    arch: string;
    code_snippet?: string;
    ai_analysis?: {
        root_cause_explanation: string;
        confidence: "high" | "medium" | "low";
        fix_steps: string[];
        bug_pattern: string;
        risk_level: "critical" | "high" | "medium";
    };
}

export class CLIWrapper {
    private static pythonPath: string = "python"; // Should be configurable

    public static async decodeHardFault(inputJson: string, elfPath?: string): Promise<HardFaultResult> {
        return new Promise((resolve, reject) => {
            // Resolve absolute paths for the bundled backend
            const extension = vscode.extensions.getExtension("hardcoreai.hardcore-ai");
            const extensionPath = extension?.extensionPath || "";
            const backendDir = path.join(extensionPath, 'backend');
            const cliPath = path.join(backendDir, "cli", "hc_fault.py");
            
            // Resolve bundled Python interpreter
            const venvDir = path.join(backendDir, '.venv');
            const pythonPath = process.platform === 'win32' 
                ? path.join(venvDir, 'Scripts', 'python.exe') 
                : path.join(venvDir, 'bin', 'python');
            
            const pythonInterpreter = fs.existsSync(pythonPath) ? pythonPath : "python";
            
            // Create a temporary file for the JSON input
            const tmp = os.tmpdir();
            const tmpFile = path.join(tmp, `hf_${Date.now()}.json`);
            fs.writeFileSync(tmpFile, inputJson);

            let args = ["debug", "hardfault", tmpFile, "--json"];
            if (elfPath) {
                args.push("--elf", elfPath);
            }

            const config = vscode.workspace.getConfiguration('hardcoreai');
            const apiKey = config.get<string>('geminiApiKey');

            const pythonProcess = cp.spawn(pythonInterpreter, [cliPath, ...args], {
                env: {
                    ...process.env,
                    GEMINI_API_KEY: apiKey || ""
                }
            });

            let stdout = "";
            let stderr = "";

            pythonProcess.stdout.on("data", (data: any) => {
                stdout += data.toString();
            });

            pythonProcess.stderr.on("data", (data: any) => {
                stderr += data.toString();
            });

            pythonProcess.on("close", (code: any) => {
                // Clean up tmp file
                if (fs.existsSync(tmpFile)) fs.unlinkSync(tmpFile);

                if (code !== 0) {
                    reject(new Error(`CLI failed with code ${code}: ${stderr}`));
                    return;
                }

                try {
                    const result = JSON.parse(stdout);
                    resolve(result);
                } catch (e) {
                    reject(new Error(`Failed to parse CLI output: ${stdout}`));
                }
            });
        });
    }
}
