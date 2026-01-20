/**
 * OpenCode NTM Integration Plugin
 * 
 * This plugin hooks into the OpenCode session to provide deep observability
 * for the NTM orchestration system. It logs events to a file that NTM can monitor,
 * or simply emits state changes.
 */

const fs = require('fs');
const path = require('path');

// Logging helper
function log(msg, data) {
    const logLine = `[NTM Plugin] ${new Date().toISOString()} ${msg} ${data ? JSON.stringify(data) : ''}\n`;
    // Try to log to a file in the project root if console is captured/swallowed
    try {
        fs.appendFileSync(path.join(process.cwd(), '.opencode', 'ntm-plugin.log'), logLine);
    } catch (e) {
        // Fallback to console
        console.log(logLine);
    }
}

module.exports = {
    /**
     * Plugin initialization.
     * @param {Object} context - The OpenCode plugin context
     */
    init: function(context) {
        log("Plugin Initialized");
        log("Context Keys available:", Object.keys(context));

        // Hook into session events if available
        if (context.events) {
            context.events.on('*', (event, ...args) => {
                log(`Event Received: ${event}`, args);
            });
            
            // Specific known events (hypothetical based on standard patterns)
            context.events.on('session.start', (session) => {
                log("Session Started", { id: session.id });
            });
            
            context.events.on('message', (msg) => {
                log("Message", { role: msg.role, content: msg.content.substring(0, 50) + "..." });
            });
            
            context.events.on('tool.call', (tool) => {
                log("Tool Call", { name: tool.name });
            });
            
            context.events.on('status', (status) => {
                log("Status Change", status);
            });
        } else {
            log("No 'events' object found in context. Inspecting context properties for other hooks...");
            // Iterate headers/services
            for (const key of Object.keys(context)) {
                if (typeof context[key] === 'function') {
                    // Potential hooks
                }
            }
        }
    }
};
