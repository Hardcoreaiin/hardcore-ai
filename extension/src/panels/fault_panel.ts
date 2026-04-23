// @ts-ignore
import * as vscode from 'vscode';
import { HardFaultResult } from '../modules/cli_wrapper';

export class FaultPanel {
    public static currentPanel: FaultPanel | undefined;
    private readonly _panel: vscode.WebviewPanel;
    private _disposables: vscode.Disposable[] = [];
    private static _onOpenSourceCallback: ((data: any) => void) | undefined;

    private constructor(panel: vscode.WebviewPanel, result: HardFaultResult) {
        this._panel = panel;
        this._update(result);
        this._panel.onDidDispose(() => this.dispose(), null, this._disposables);

        // Handle messages from the webview
        this._panel.webview.onDidReceiveMessage(
            message => {
                switch (message.command) {
                    case 'openSource':
                        if (FaultPanel._onOpenSourceCallback) {
                            FaultPanel._onOpenSourceCallback(message.data);
                        }
                        return;
                }
            },
            null,
            this._disposables
        );
    }

    public static createOrShow(result: HardFaultResult) {
        const column = vscode.window.activeTextEditor ? vscode.window.activeTextEditor.viewColumn : undefined;

        if (FaultPanel.currentPanel) {
            FaultPanel.currentPanel._panel.reveal(column);
            FaultPanel.currentPanel._update(result);
            return;
        }

        const panel = vscode.window.createWebviewPanel(
            'hardfaultResult',
            'HARDCOREAI: Diagnostic Dashboard',
            column || vscode.ViewColumn.Two,
            {
                enableScripts: true,
                retainContextWhenHidden: true
            }
        );

        FaultPanel.currentPanel = new FaultPanel(panel, result);
    }

    public static onOpenSource(callback: (data: any) => void) {
        this._onOpenSourceCallback = callback;
    }

    public dispose() {
        FaultPanel.currentPanel = undefined;
        this._panel.dispose();
        while (this._disposables.length) {
            const x = this._disposables.pop();
            if (x) {
                x.dispose();
            }
        }
    }

    private _update(result: HardFaultResult) {
        this._panel.webview.html = this._getHtmlForWebview(result);
    }

    private _getHtmlForWebview(result: HardFaultResult) {
        const confidenceClass = result.confidence.toLowerCase();
        
        // Helper to generate register badges
        const detailsHtml = result.raw_details.map(d => `
            <div class="reg-mini-card">
                <span class="reg-name">${d.register}</span>
                <span class="reg-val">${d.description}</span>
            </div>
        `).join('');

        const fixHtml = result.fix.map(step => `
            <div class="fix-item">
                <div class="fix-check"></div>
                <div class="fix-text">${step}</div>
            </div>
        `).join('');

        const evidenceHtml = (result as any).evidence && (result as any).evidence.length > 0 ? `
            <div class="glass-card full-span">
                <div class="card-label"><i class="codicon codicon-verified"></i> Diagnostic Evidence</div>
                <div class="fact-grid">
                    ${(result as any).evidence.map((fact: string) => `<div class="fact-tag">${fact}</div>`).join('')}
                </div>
            </div>
        ` : '';

        const timelineHtml = (result as any).timeline && (result as any).timeline.length > 0 ? `
            <div class="glass-card">
                <div class="card-label"><i class="codicon codicon-history"></i> Fault Timeline</div>
                <div class="timeline-v-line">
                    ${(result as any).timeline.map((note: string) => `
                        <div class="timeline-point">
                            <div class="t-dot"></div>
                            <div class="t-msg">${note}</div>
                        </div>
                    `).join('')}
                </div>
            </div>
        ` : '';

        const aiAnalysisHtml = result.ai_analysis ? `
            <div class="glass-card ai-intelligence glowing">
                <div class="ai-badge">NEURAL ENGINE ACTIVE</div>
                <div class="card-label" style="color: #b388ff;">🧠 AI ANALYSIS & REASONING</div>
                
                <h3 class="ai-pattern-title">${result.ai_analysis.bug_pattern}</h3>
                <p class="ai-explanation-text">${result.ai_analysis.root_cause_explanation}</p>
                
                <div class="card-label small">AI FIX PROTOCOL</div>
                <div class="ai-fix-grid">
                    ${result.ai_analysis.fix_steps.map(step => `
                        <div class="ai-fix-bubble">${step}</div>
                    `).join('')}
                </div>

                <div class="ai-footer-bar">
                    <div class="ai-conf-meter">
                        <div class="ai-conf-fill" style="width: ${result.ai_analysis.confidence === 'high' ? '95' : result.ai_analysis.confidence === 'medium' ? '65' : '35'}%"></div>
                    </div>
                    <span>CONFIDENCE: ${result.ai_analysis.confidence.toUpperCase()}</span>
                </div>
            </div>
        ` : `
            <div class="glass-card empty-ai">
                <div class="card-label">🧠 AI INTELLIGENCE</div>
                <div class="loader-dots"><span></span><span></span><span></span></div>
                <p>Waiting for neural reasoning layer...</p>
            </div>
        `;

        return `<!DOCTYPE html>
        <html lang="en">
        <head>
            <meta charset="UTF-8">
            <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;600;800&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
            <style>
                :root {
                    --bg: #0b0e14;
                    --fg: #e1e4e8;
                    --accent: #4fc3f7;
                    --accent-deep: #0288d1;
                    --error: #ff5252;
                    --success: #00e676;
                    --warning: #ffab40;
                    --ai-border: #6200ea;
                    --glass: rgba(255, 255, 255, 0.03);
                    --glass-border: rgba(255, 255, 255, 0.1);
                    --font-main: 'Inter', system-ui, -apple-system, sans-serif;
                    --font-mono: 'Fira Code', monospace;
                }

                @keyframes glow {
                    0% { box-shadow: 0 0 5px rgba(98, 0, 234, 0.2); }
                    50% { box-shadow: 0 0 20px rgba(98, 0, 234, 0.4); }
                    100% { box-shadow: 0 0 5px rgba(98, 0, 234, 0.2); }
                }

                @keyframes fadeInUp {
                    from { opacity: 0; transform: translateY(10px); }
                    to { opacity: 1; transform: translateY(0); }
                }

                body { 
                    font-family: var(--font-main); 
                    background: var(--bg); 
                    color: var(--fg); 
                    margin: 0; 
                    padding: 40px; 
                    line-height: 1.6;
                    overflow-x: hidden;
                }

                .dashboard {
                    max-width: 1400px;
                    margin: 0 auto;
                    animation: fadeInUp 0.5s ease-out;
                }

                /* Header Area */
                header {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    margin-bottom: 40px;
                    padding-bottom: 20px;
                    border-bottom: 1px solid var(--glass-border);
                }

                .brand h1 {
                    margin: 0;
                    font-weight: 800;
                    font-size: 1.5rem;
                    letter-spacing: -0.5px;
                    background: linear-gradient(90deg, #fff, var(--accent));
                    -webkit-background-clip: text;
                    -webkit-text-fill-color: transparent;
                }

                .brand span {
                    font-size: 0.65rem;
                    text-transform: uppercase;
                    letter-spacing: 2px;
                    opacity: 0.5;
                }

                .status-badge {
                    padding: 8px 16px;
                    border-radius: 6px;
                    font-size: 0.75rem;
                    font-weight: 600;
                    text-transform: uppercase;
                    letter-spacing: 1px;
                    border: 1px solid var(--glass-border);
                    background: var(--glass);
                }
                .status-deterministic { color: var(--success); border-color: var(--success); }
                .status-high { color: var(--accent); border-color: var(--accent); }
                .status-medium { color: var(--warning); border-color: var(--warning); }
                .status-low { color: var(--error); border-color: var(--error); }

                /* Glance Bar */
                .glance-bar {
                    display: grid;
                    grid-template-columns: repeat(4, 1fr);
                    gap: 20px;
                    margin-bottom: 30px;
                }

                .glance-item {
                    background: var(--glass);
                    border: 1px solid var(--glass-border);
                    padding: 15px 20px;
                    border-radius: 12px;
                }

                .glance-label { font-size: 0.6rem; text-transform: uppercase; opacity: 0.5; margin-bottom: 4px; font-weight: 700; }
                .glance-value { font-family: var(--font-mono); font-size: 0.9rem; color: var(--accent); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

                /* Main Grid */
                .main-grid {
                    display: grid;
                    grid-template-columns: 1.5fr 1fr;
                    gap: 30px;
                }

                .glass-card {
                    background: var(--glass);
                    backdrop-filter: blur(20px);
                    border: 1px solid var(--glass-border);
                    border-radius: 16px;
                    padding: 30px;
                    margin-bottom: 30px;
                    transition: transform 0.2s, background 0.2s;
                }
                .glass-card:hover { background: rgba(255, 255, 255, 0.05); }

                .card-label {
                    font-size: 0.7rem;
                    font-weight: 800;
                    text-transform: uppercase;
                    letter-spacing: 1.5px;
                    margin-bottom: 20px;
                    opacity: 0.7;
                    display: flex;
                    align-items: center;
                    gap: 8px;
                }

                .primary-fault-type {
                    font-size: 2.2rem;
                    margin: 0 0 10px 0;
                    font-weight: 800;
                    color: var(--error);
                    line-height: 1;
                }

                .main-description {
                    font-size: 1.05rem;
                    color: #bbb;
                    margin-bottom: 25px;
                    max-width: 600px;
                }

                /* Source & Code Snippet */
                .source-anchor {
                    display: inline-flex;
                    align-items: center;
                    background: rgba(79, 195, 247, 0.1);
                    padding: 8px 12px;
                    border-radius: 8px;
                    color: var(--accent);
                    text-decoration: none;
                    font-family: var(--font-mono);
                    font-size: 0.85rem;
                    cursor: pointer;
                    margin-bottom: 20px;
                }

                .code-display {
                    background: #000;
                    border-radius: 12px;
                    padding: 20px;
                    font-family: var(--font-mono);
                    font-size: 0.85rem;
                    border: 1px solid #222;
                    overflow-x: auto;
                }
                .line-active { background: rgba(255, 82, 82, 0.15); border-left: 2px solid var(--error); margin: 0 -20px; padding: 0 20px; color: #fff; }

                /* AI Section */
                .ai-intelligence {
                    background: linear-gradient(135deg, rgba(98, 0, 234, 0.15), rgba(15, 12, 41, 0.5));
                    border-color: rgba(98, 0, 234, 0.3) !important;
                }
                .glowing { animation: glow 3s infinite ease-in-out; }

                .ai-badge {
                    display: inline-block;
                    background: #6200ea;
                    color: #fff;
                    font-size: 0.6rem;
                    font-weight: 900;
                    padding: 3px 8px;
                    border-radius: 4px;
                    margin-bottom: 12px;
                }

                .ai-pattern-title { font-size: 1.4rem; margin: 0 0 15px 0; color: #fff; font-weight: 600; }
                .ai-explanation-text { font-size: 1rem; color: #ccc; line-height: 1.7; margin-bottom: 25px; }

                .ai-fix-grid { display: flex; flex-wrap: wrap; gap: 10px; margin-bottom: 30px; }
                .ai-fix-bubble { background: rgba(98, 0, 234, 0.2); border: 1px solid rgba(179, 136, 255, 0.3); padding: 8px 16px; border-radius: 40px; font-size: 0.8rem; color: #d1c4e9; }

                .ai-footer-bar {
                    display: flex;
                    align-items: center;
                    gap: 15px;
                    margin-top: 30px;
                    padding-top: 20px;
                    border-top: 1px solid rgba(255,255,255,0.05);
                    font-size: 0.7rem;
                    font-weight: 800;
                }
                .ai-conf-meter { flex-grow: 1; height: 4px; background: rgba(255,255,255,0.05); border-radius: 2px; }
                .ai-conf-fill { height: 100%; background: #6200ea; border-radius: 2px; }

                /* Timeline */
                .timeline-v-line { border-left: 2px solid var(--glass-border); padding-left: 25px; margin-left: 10px; }
                .timeline-point { position: relative; margin-bottom: 25px; }
                .t-dot { position: absolute; left: -31px; top: 8px; width: 10px; height: 10px; background: var(--accent); border-radius: 50%; border: 3px solid var(--bg); }
                .t-msg { font-size: 0.95rem; color: #ddd; }

                /* Fact Matrix */
                .fact-grid { display: flex; flex-wrap: wrap; gap: 10px; }
                .fact-tag { background: rgba(255,255,255,0.04); border: 1px solid var(--glass-border); padding: 6px 12px; border-radius: 6px; font-size: 0.8rem; color: #aaa; }

                /* Fix Engine */
                .fix-engine { border: 1px solid var(--success); background: linear-gradient(135deg, rgba(0, 230, 118, 0.05), transparent); }
                .fix-item { display: flex; align-items: flex-start; gap: 15px; margin-bottom: 20px; }
                .fix-check { width: 18px; height: 18px; border: 2px solid var(--success); border-radius: 50%; margin-top: 4px; flex-shrink: 0; }
                .fix-text { font-size: 0.95rem; color: #fff; }

                /* Register Summary */
                .reg-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 10px; }
                .reg-mini-card { background: rgba(0,0,0,0.2); padding: 12px; border-radius: 8px; border: 1px solid var(--glass-border); }
                .reg-name { display: block; font-size: 0.55rem; font-weight: 900; opacity: 0.4; margin-bottom: 4px; }
                .reg-val { font-family: var(--font-mono); font-size: 0.85rem; color: var(--accent); }

                /* Stack View */
                .stack-table { margin-top: 15px; width: 100%; }
                .stack-row { display: grid; grid-template-columns: 100px 1fr 120px; padding: 12px; border-bottom: 1px solid var(--glass-border); cursor: pointer; transition: background 0.2s; border-radius: 4px; }
                .stack-row:hover { background: rgba(255,255,255,0.05); }
                .stack-cell { font-size: 0.85rem; }
                .st-addr { font-family: var(--font-mono); opacity: 0.5; }
                .st-func { color: var(--accent); font-weight: 600; }
                .st-file { text-align: right; opacity: 0.4; font-size: 0.75rem; }

                .empty-ai { opacity: 0.5; border-style: dashed !important; border-width: 2px !important; text-align: center; padding: 60px 40px; }
                .loader-dots { display: flex; justify-content: center; gap: 8px; margin-bottom: 15px; }
                .loader-dots span { width: 8px; height: 8px; background: #fff; border-radius: 50%; opacity: 0.2; animation: loaderPulse 1s infinite alternate; }
                .loader-dots span:nth-child(2) { animation-delay: 0.3s; }
                .loader-dots span:nth-child(3) { animation-delay: 0.6s; }
                @keyframes loaderPulse { from { opacity: 0.2; } to { opacity: 0.8; } }
            </style>
        </head>
        <body>
            <div class="dashboard">
                <header>
                    <div class="brand">
                        <h1>HARDCOREAI // 01</h1>
                        <span>UNIVERSAL CRASH INTELLIGENCE ENGINE</span>
                    </div>
                    <div class="status-badge status-${confidenceClass}">
                        ${result.confidence} Resolution
                    </div>
                </header>

                <div class="glance-bar">
                    <div class="glance-item">
                        <div class="glance-label">Program Counter</div>
                        <div class="glance-value">${result.where.address}</div>
                    </div>
                    <div class="glance-item">
                        <div class="glance-label">Return Address</div>
                        <div class="glance-value">${result.raw_registers.lr ? (typeof result.raw_registers.lr === 'number' ? '0x' + result.raw_registers.lr.toString(16) : result.raw_registers.lr) : 'N/A'}</div>
                    </div>
                    <div class="glance-item">
                        <div class="glance-label">Architecture</div>
                        <div class="glance-value">${result.arch}</div>
                    </div>
                    <div class="glance-item">
                        <div class="glance-label">Fault Location</div>
                        <div class="glance-value">${result.where.function}()</div>
                    </div>
                </div>

                <div class="main-grid">
                    <div class="left-col">
                        <div class="glass-card">
                            <div class="card-label">Identity</div>
                            <h2 class="primary-fault-type">${result.fault_type}</h2>
                            <p class="main-description">${result.what_happened}</p>
                            
                            <div class="source-anchor" onclick="openSource('${result.where.file}', ${result.where.line}, '${(result.where as any).fullPath || ''}')">
                                → CRASH SITE: ${result.where.file}:${result.where.line}
                            </div>

                            ${result.code_snippet ? `
                            <div class="code-display">
                                ${result.code_snippet.split('\n').map((line: string) => {
                                    if (line.includes('→')) return `<div class="line-active">${line}</div>`;
                                    return `<div>${line}</div>`;
                                }).join('')}
                            </div>
                            ` : ''}
                        </div>

                        ${aiAnalysisHtml}
                        ${evidenceHtml}
                    </div>

                    <div class="right-col">
                        <div class="glass-card fix-engine">
                            <div class="card-label" style="color: var(--success);">Actionable Fix Engine</div>
                            <div class="fix-list-container">
                                ${fixHtml}
                            </div>
                        </div>

                        ${timelineHtml}

                        <div class="glass-card">
                            <div class="card-label">Register Snapshot</div>
                            <div class="reg-grid">
                                ${detailsHtml}
                            </div>
                        </div>
                        
                        ${(result as any).stack_trace && (result as any).stack_trace.length > 0 ? `
                        <div class="glass-card">
                            <div class="card-label">Call Stack (Reconstructed)</div>
                            <div class="stack-table">
                                ${(result as any).stack_trace.map((entry: any) => `
                                    <div class="stack-row" onclick="openSource('${entry.file}', ${entry.line}, '${entry.fullPath || ''}')">
                                        <div class="stack-cell st-addr">${entry.address}</div>
                                        <div class="stack-cell st-func">${entry.function}()</div>
                                        <div class="stack-cell st-file">${entry.file}:${entry.line}</div>
                                    </div>
                                `).join('')}
                            </div>
                        </div>
                        ` : ''}
                    </div>
                </div>
            </div>

            <script>
                const vscode = acquireVsCodeApi();
                function openSource(file, line, fullPath) {
                    vscode.postMessage({
                        command: 'openSource',
                        data: { file, line, fullPath }
                    });
                }
            </script>
        </body>
        </html>`;
    }
}
