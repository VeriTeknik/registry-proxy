// Check authentication on page load
document.addEventListener('DOMContentLoaded', () => {
    checkAuth();
    loadServers();
});

// Global variables
let currentServerId = null;
let serversTable = null;
let mcpImporter = null;
let importedServers = [];

// Authentication functions
async function checkAuth() {
    const token = localStorage.getItem('token');
    const tokenExpiry = localStorage.getItem('tokenExpiry');
    
    if (!token || !tokenExpiry || new Date().getTime() > tokenExpiry * 1000) {
        window.location.href = '/login';
        return;
    }
    
    try {
        const response = await fetch('/api/auth/verify', {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        if (!response.ok) {
            window.location.href = '/login';
        }
    } catch (error) {
        window.location.href = '/login';
    }
}

function logout() {
    localStorage.removeItem('token');
    localStorage.removeItem('tokenExpiry');
    window.location.href = '/login';
}

// API helper function
async function apiRequest(url, options = {}) {
    const token = localStorage.getItem('token');
    const defaultOptions = {
        headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
        }
    };
    
    const response = await fetch(url, { ...defaultOptions, ...options });
    
    if (response.status === 401) {
        window.location.href = '/login';
        throw new Error('Unauthorized');
    }
    
    return response;
}

// Server management functions
async function loadServers() {
    try {
        const response = await apiRequest('/api/servers?limit=1000');
        const data = await response.json();
        
        // Update stats
        const servers = data.servers || [];
        document.getElementById('total-servers').textContent = servers.length;
        document.getElementById('active-servers').textContent = 
            servers.filter(s => !s.status || s.status === 'active').length;
        document.getElementById('deprecated-servers').textContent = 
            servers.filter(s => s.status === 'deprecated').length;
        
        // Initialize or update DataTable
        if (serversTable) {
            serversTable.clear();
            serversTable.rows.add(servers.map(formatServerRow));
            serversTable.draw();
        } else {
            serversTable = $('#servers-table').DataTable({
                data: servers.map(formatServerRow),
                columns: [
                    { title: 'Name' },
                    { title: 'Description' },
                    { title: 'Status' },
                    { title: 'Version' },
                    { title: 'Registry' },
                    { title: 'Actions' }
                ],
                pageLength: 25,
                order: [[0, 'asc']]
            });
        }
    } catch (error) {
        console.error('Failed to load servers:', error);
    }
}

function formatServerRow(server) {
    const statusBadge = (!server.status || server.status === 'active')
        ? '<span class="px-2 py-1 text-xs rounded-full bg-green-100 text-green-800">Active</span>'
        : '<span class="px-2 py-1 text-xs rounded-full bg-yellow-100 text-yellow-800">Deprecated</span>';
    
    const registryName = server.packages && server.packages[0] 
        ? server.packages[0].registry_name 
        : 'N/A';
    
    const actions = `
        <button onclick="editServer('${server.id}')" class="text-blue-600 hover:text-blue-900 mr-2">Edit</button>
        <button onclick="toggleStatus('${server.id}', '${server.status || 'active'}')" class="text-yellow-600 hover:text-yellow-900 mr-2">Toggle</button>
        <button onclick="deleteServer('${server.id}')" class="text-red-600 hover:text-red-900">Delete</button>
    `;
    
    return [
        server.name || 'N/A',
        (server.description || '').substring(0, 50) + (server.description?.length > 50 ? '...' : ''),
        statusBadge,
        server.version_detail?.version || 'N/A',
        registryName,
        actions
    ];
}

// Modal functions
function showAddServerModal() {
    currentServerId = null;
    document.getElementById('modal-title').textContent = 'Add Server';
    document.getElementById('server-form').reset();
    document.getElementById('server-modal').classList.remove('hidden');
}

async function editServer(id) {
    currentServerId = id;
    document.getElementById('modal-title').textContent = 'Edit Server';
    
    try {
        const response = await apiRequest(`/api/servers/${id}`);
        const server = await response.json();
        
        document.getElementById('server-name').value = server.name || '';
        document.getElementById('server-description').value = server.description || '';
        document.getElementById('server-status').value = server.status || 'active';
        document.getElementById('server-repo-url').value = server.repository?.url || '';
        document.getElementById('server-version').value = server.version_detail?.version || '';
        document.getElementById('server-packages').value = JSON.stringify(server.packages || [], null, 2);
        
        document.getElementById('server-modal').classList.remove('hidden');
    } catch (error) {
        alert('Failed to load server details');
    }
}

function closeModal() {
    document.getElementById('server-modal').classList.add('hidden');
    currentServerId = null;
}

// Form submission
document.getElementById('server-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const serverData = {
        name: document.getElementById('server-name').value,
        description: document.getElementById('server-description').value,
        status: document.getElementById('server-status').value,
        repository: {
            url: document.getElementById('server-repo-url').value,
            source: 'github'
        },
        version_detail: {
            version: document.getElementById('server-version').value,
            release_date: new Date().toISOString(),
            is_latest: true
        }
    };
    
    // Parse packages JSON
    const packagesText = document.getElementById('server-packages').value;
    if (packagesText) {
        try {
            serverData.packages = JSON.parse(packagesText);
        } catch (error) {
            alert('Invalid JSON in packages field');
            return;
        }
    }
    
    try {
        const url = currentServerId ? `/api/servers/${currentServerId}` : '/api/servers';
        const method = currentServerId ? 'PUT' : 'POST';
        
        const response = await apiRequest(url, {
            method: method,
            body: JSON.stringify(serverData)
        });
        
        if (response.ok) {
            closeModal();
            loadServers();
        } else {
            const error = await response.text();
            alert(`Failed to save server: ${error}`);
        }
    } catch (error) {
        alert('Failed to save server');
    }
});

// Action functions
async function toggleStatus(id, currentStatus) {
    const newStatus = (!currentStatus || currentStatus === 'active') ? 'deprecated' : 'active';
    
    try {
        const response = await apiRequest(`/api/servers/${id}/status`, {
            method: 'PATCH',
            body: JSON.stringify({ status: newStatus })
        });
        
        if (response.ok) {
            loadServers();
        } else {
            alert('Failed to update status');
        }
    } catch (error) {
        alert('Failed to update status');
    }
}

async function deleteServer(id) {
    if (!confirm('Are you sure you want to delete this server?')) {
        return;
    }
    
    try {
        const response = await apiRequest(`/api/servers/${id}`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            loadServers();
        } else {
            alert('Failed to delete server');
        }
    } catch (error) {
        alert('Failed to delete server');
    }
}

// Audit logs
async function showAuditLogs() {
    try {
        const response = await apiRequest('/api/audit-logs?limit=50');
        const logs = await response.json();
        
        const tbody = document.getElementById('audit-logs-body');
        tbody.innerHTML = '';
        
        logs.forEach(log => {
            const row = tbody.insertRow();
            row.innerHTML = `
                <td class="p-2">${new Date(log.timestamp).toLocaleString()}</td>
                <td class="p-2">${log.user}</td>
                <td class="p-2">${log.action}</td>
                <td class="p-2">${log.details}</td>
            `;
        });
        
        document.getElementById('audit-modal').classList.remove('hidden');
    } catch (error) {
        alert('Failed to load audit logs');
    }
}

function closeAuditModal() {
    document.getElementById('audit-modal').classList.add('hidden');
}

// MCP Config Import functionality
class MCPConfigImporter {
    constructor() {
        this.registryMappings = {
            'npx': { registry: 'npm', hint: 'npx', extract: this.extractNpmPackage },
            'npm': { registry: 'npm', hint: 'npx', extract: this.extractNpmPackage },
            'node': { registry: 'npm', hint: null, extract: this.extractNodePackage },
            'uvx': { registry: 'pypi', hint: 'uvx', extract: this.extractPypiPackage },
            'uv': { registry: 'pypi', hint: 'uvx', extract: this.extractPypiPackage },
            'pip': { registry: 'pypi', hint: null, extract: this.extractPypiPackage },
            'python': { registry: 'pypi', hint: null, extract: this.extractPythonPackage },
            'python3': { registry: 'pypi', hint: null, extract: this.extractPythonPackage },
            'docker': { registry: 'docker', hint: 'docker', extract: this.extractDockerPackage },
            'dotnet': { registry: 'nuget', hint: 'dnx', extract: this.extractNugetPackage }
        };
    }

    extractNpmPackage(command, args) {
        let packageName = null;
        let packageArgs = [];
        let skipNext = false;
        
        for (let i = 0; i < args.length; i++) {
            if (skipNext) {
                skipNext = false;
                continue;
            }
            
            const arg = args[i];
            
            // Skip npx/npm flags
            if (arg === '-y' || arg === '--yes' || arg === '-p' || arg === '--package') {
                if (arg === '-p' || arg === '--package') {
                    skipNext = true;
                }
                continue;
            }
            
            // First non-flag is package name
            if (!packageName && !arg.startsWith('-')) {
                packageName = arg;
                continue;
            }
            
            // Rest are package arguments
            if (arg.startsWith('--')) {
                const next = args[i + 1];
                if (next && !next.startsWith('-')) {
                    packageArgs.push({
                        type: 'named',
                        name: arg,
                        value: next,
                        description: `${arg} parameter`
                    });
                    skipNext = true;
                } else {
                    packageArgs.push({
                        type: 'named',
                        name: arg,
                        value: 'true'
                    });
                }
            } else if (!arg.startsWith('-')) {
                packageArgs.push({
                    type: 'positional',
                    value: arg,
                    description: 'Path or argument',
                    value_hint: arg.includes('/') || arg.includes('\\') ? 'path' : undefined
                });
            }
        }
        
        return { packageName, packageArgs };
    }

    extractNodePackage(command, args) {
        // For node commands, first arg is usually the script
        const packageName = args[0];
        const packageArgs = args.slice(1).map(arg => ({
            type: 'positional',
            value: arg
        }));
        return { packageName, packageArgs };
    }

    extractPypiPackage(command, args) {
        // Handle uvx/uv/pip patterns
        let packageName = null;
        let packageArgs = [];
        
        for (let i = 0; i < args.length; i++) {
            const arg = args[i];
            
            if (!packageName && !arg.startsWith('-')) {
                packageName = arg;
            } else if (!arg.startsWith('-')) {
                packageArgs.push({
                    type: 'positional',
                    value: arg
                });
            }
        }
        
        return { packageName, packageArgs };
    }

    extractPythonPackage(command, args) {
        // For python -m module pattern
        if (args[0] === '-m' && args[1]) {
            return {
                packageName: args[1],
                packageArgs: args.slice(2).map(arg => ({
                    type: 'positional',
                    value: arg
                }))
            };
        }
        return this.extractPypiPackage(command, args);
    }

    extractDockerPackage(command, args) {
        let packageName = null;
        let runtimeArgs = [];
        let packageArgs = [];
        
        const isRunCommand = args[0] === 'run';
        const startIdx = isRunCommand ? 1 : 0;
        
        for (let i = startIdx; i < args.length; i++) {
            const arg = args[i];
            
            if (!packageName && !arg.startsWith('-')) {
                packageName = arg;
                continue;
            }
            
            // Docker-specific runtime arguments
            if (arg === '-v' || arg === '--volume') {
                const volumeSpec = args[i + 1];
                if (volumeSpec) {
                    runtimeArgs.push({
                        type: 'named',
                        name: '--mount',
                        value: `type=bind,src={source_path},dst={target_path}`,
                        variables: {
                            source_path: {
                                description: 'Source path on host',
                                format: 'filepath',
                                is_required: true
                            },
                            target_path: {
                                description: 'Target path in container',
                                default: '/data'
                            }
                        }
                    });
                    i++;
                }
            } else if (packageName && !arg.startsWith('-')) {
                packageArgs.push({
                    type: 'positional',
                    value: arg
                });
            }
        }
        
        return { packageName, runtimeArgs, packageArgs };
    }

    extractNugetPackage(command, args) {
        // For dotnet tool run pattern
        const packageName = args[0];
        const packageArgs = args.slice(1).map(arg => ({
            type: 'positional',
            value: arg
        }));
        return { packageName, packageArgs };
    }

    convertEnvVars(env) {
        if (!env) return [];
        
        return Object.entries(env).map(([name, value]) => {
            const envVar = {
                name,
                description: this.generateEnvDescription(name),
                is_required: true,
                is_secret: this.isSecretVar(name)
            };
            
            // Only add default if it's not a placeholder
            if (value && !value.includes('${input:')) {
                envVar.default = value;
            }
            
            return envVar;
        });
    }

    generateEnvDescription(name) {
        const descriptions = {
            'API_KEY': 'API key for authentication',
            'TOKEN': 'Authentication token',
            'SECRET': 'Secret value',
            'PASSWORD': 'Password for authentication',
            'LOG_LEVEL': 'Logging level (debug, info, warn, error)',
            'PORT': 'Port number',
            'HOST': 'Host address',
            'DATABASE_URL': 'Database connection string'
        };
        
        for (const [key, desc] of Object.entries(descriptions)) {
            if (name.includes(key)) return desc;
        }
        
        return `${name} environment variable`;
    }

    isSecretVar(name) {
        const secretPatterns = ['KEY', 'TOKEN', 'SECRET', 'PASSWORD', 'CREDENTIAL'];
        return secretPatterns.some(pattern => name.toUpperCase().includes(pattern));
    }

    generateServerName(packageName, serverKey) {
        // Try to generate a compliant server name
        const cleanName = packageName
            .replace(/^@/, '')
            .replace('/', '-')
            .toLowerCase();
        
        // Default to a generic pattern
        return `io.github.user/${cleanName || serverKey}`;
    }

    parseConfig(configJson) {
        try {
            const config = JSON.parse(configJson);
            if (!config.mcpServers || typeof config.mcpServers !== 'object') {
                throw new Error('Invalid MCP config: missing mcpServers object');
            }
            return config;
        } catch (error) {
            throw new Error(`Failed to parse JSON: ${error.message}`);
        }
    }

    convertToServers(mcpConfig) {
        const servers = [];
        
        for (const [serverKey, serverConfig] of Object.entries(mcpConfig.mcpServers)) {
            const mapping = this.registryMappings[serverConfig.command];
            if (!mapping) {
                console.warn(`Unknown command '${serverConfig.command}' for server '${serverKey}'`);
                continue;
            }
            
            const extracted = mapping.extract.call(this, serverConfig.command, serverConfig.args || []);
            if (!extracted.packageName) {
                console.warn(`Could not extract package name for '${serverKey}'`);
                continue;
            }
            
            const server = {
                key: serverKey,
                name: this.generateServerName(extracted.packageName, serverKey),
                description: `${serverKey} MCP server`,
                status: 'active',
                repository: {
                    url: '',
                    source: 'github'
                },
                version_detail: {
                    version: '1.0.0',
                    release_date: new Date().toISOString(),
                    is_latest: true
                },
                packages: [{
                    registry_name: mapping.registry,
                    name: extracted.packageName,
                    version: 'latest',
                    ...(mapping.hint && { runtime_hint: mapping.hint }),
                    ...(extracted.runtimeArgs?.length && { runtime_arguments: extracted.runtimeArgs }),
                    ...(extracted.packageArgs?.length && { package_arguments: extracted.packageArgs }),
                    environment_variables: this.convertEnvVars(serverConfig.env)
                }]
            };
            
            servers.push(server);
        }
        
        return servers;
    }
}

// Import modal functions
function showImportModal() {
    document.getElementById('import-modal').classList.remove('hidden');
    document.getElementById('import-step-1').classList.remove('hidden');
    document.getElementById('import-step-2').classList.add('hidden');
    document.getElementById('import-step-3').classList.add('hidden');
    document.getElementById('import-config-input').value = '';
    document.getElementById('import-parse-error').classList.add('hidden');
    
    if (!mcpImporter) {
        mcpImporter = new MCPConfigImporter();
    }
}

function closeImportModal() {
    document.getElementById('import-modal').classList.add('hidden');
    importedServers = [];
}

function parseImportConfig() {
    const configInput = document.getElementById('import-config-input').value.trim();
    const errorDiv = document.getElementById('import-parse-error');
    
    if (!configInput) {
        errorDiv.textContent = 'Please paste a configuration';
        errorDiv.classList.remove('hidden');
        return;
    }
    
    try {
        const config = mcpImporter.parseConfig(configInput);
        importedServers = mcpImporter.convertToServers(config);
        
        if (importedServers.length === 0) {
            errorDiv.textContent = 'No valid servers found in configuration';
            errorDiv.classList.remove('hidden');
            return;
        }
        
        // Show step 2 with converted servers
        displayImportedServers();
        document.getElementById('import-step-1').classList.add('hidden');
        document.getElementById('import-step-2').classList.remove('hidden');
        errorDiv.classList.add('hidden');
        
    } catch (error) {
        errorDiv.textContent = error.message;
        errorDiv.classList.remove('hidden');
    }
}

function displayImportedServers() {
    const container = document.getElementById('import-servers-list');
    container.innerHTML = '';
    
    importedServers.forEach((server, index) => {
        const card = document.createElement('div');
        card.className = 'p-4 border rounded bg-gray-50';
        card.innerHTML = `
            <div class="flex items-start justify-between mb-2">
                <div>
                    <input type="checkbox" id="import-select-${index}" checked class="mr-2">
                    <label for="import-select-${index}" class="font-semibold">${server.key}</label>
                </div>
                <span class="px-2 py-1 text-xs rounded bg-blue-100 text-blue-800">
                    ${server.packages[0].registry_name}
                </span>
            </div>
            <div class="grid grid-cols-1 gap-2 text-sm">
                <div>
                    <label class="block text-gray-600">Name:</label>
                    <input type="text" 
                           id="import-name-${index}" 
                           value="${server.name}"
                           class="w-full px-2 py-1 border rounded">
                </div>
                <div>
                    <label class="block text-gray-600">Description: <span class="text-red-500">*</span></label>
                    <input type="text" 
                           id="import-desc-${index}" 
                           value="${server.description}"
                           placeholder="Enter server description"
                           class="w-full px-2 py-1 border rounded">
                </div>
                <div>
                    <label class="block text-gray-600">Repository URL: <span class="text-red-500">*</span></label>
                    <input type="url" 
                           id="import-repo-${index}" 
                           value="${server.repository.url}"
                           placeholder="https://github.com/owner/repo"
                           class="w-full px-2 py-1 border rounded">
                </div>
                <div>
                    <label class="block text-gray-600">Version:</label>
                    <input type="text" 
                           id="import-version-${index}" 
                           value="${server.version_detail.version}"
                           class="w-full px-2 py-1 border rounded">
                </div>
                <div>
                    <label class="block text-gray-600">Package:</label>
                    <div class="px-2 py-1 bg-gray-100 rounded font-mono text-xs">
                        ${server.packages[0].name} (${server.packages[0].version})
                    </div>
                </div>
                ${server.packages[0].environment_variables?.length ? `
                <div>
                    <label class="block text-gray-600">Environment Variables:</label>
                    <div class="px-2 py-1 bg-gray-100 rounded text-xs">
                        ${server.packages[0].environment_variables.map(v => v.name).join(', ')}
                    </div>
                </div>
                ` : ''}
            </div>
        `;
        container.appendChild(card);
    });
}

function backToImportStep1() {
    document.getElementById('import-step-1').classList.remove('hidden');
    document.getElementById('import-step-2').classList.add('hidden');
}

// Sync functionality
async function showSyncModal() {
    document.getElementById('sync-modal').classList.remove('hidden');
    // Reset modal state
    document.getElementById('sync-results').classList.add('hidden');
    document.getElementById('sync-error').classList.add('hidden');
    document.getElementById('sync-preview-btn').classList.remove('hidden');
    document.getElementById('sync-execute-btn').classList.add('hidden');
    document.getElementById('sync-dry-run').checked = true;
}

function closeSyncModal() {
    document.getElementById('sync-modal').classList.add('hidden');
}

async function previewSync() {
    const dryRun = document.getElementById('sync-dry-run').checked;
    const updateExisting = document.getElementById('sync-update-existing').checked;
    const addNew = document.getElementById('sync-add-new').checked;

    const button = document.getElementById('sync-preview-btn');
    const originalText = button.innerText;
    button.disabled = true;
    button.innerText = 'Loading...';

    try {
        const response = await apiRequest('/api/sync/preview', {
            method: 'POST',
            body: JSON.stringify({
                dry_run: true,
                update_existing: updateExisting,
                add_new: addNew
            })
        });

        if (response.ok) {
            const result = await response.json();
            displaySyncResults(result);

            // Show execute button if not in dry-run mode
            if (!dryRun) {
                document.getElementById('sync-execute-btn').classList.remove('hidden');
            }
        } else {
            const error = await response.text();
            showSyncError(`Failed to preview sync: ${error}`);
        }
    } catch (error) {
        showSyncError(`Error: ${error.message}`);
    } finally {
        button.disabled = false;
        button.innerText = originalText;
    }
}

async function executeSync() {
    const updateExisting = document.getElementById('sync-update-existing').checked;
    const addNew = document.getElementById('sync-add-new').checked;

    const button = document.getElementById('sync-execute-btn');
    const originalText = button.innerText;
    button.disabled = true;
    button.innerText = 'Syncing...';

    // Show progress
    document.getElementById('sync-progress').classList.remove('hidden');
    updateSyncProgress(0, 'Starting sync...');

    try {
        const response = await apiRequest('/api/sync/execute', {
            method: 'POST',
            body: JSON.stringify({
                dry_run: false,
                update_existing: updateExisting,
                add_new: addNew
            })
        });

        if (response.ok) {
            const result = await response.json();
            updateSyncProgress(100, 'Sync completed!');

            // Refresh the servers table
            await loadServers();

            // Update last sync time
            document.getElementById('last-sync-time').innerText = new Date().toLocaleString();
            document.getElementById('sync-status').classList.remove('hidden');

            setTimeout(() => {
                closeSyncModal();
                document.getElementById('sync-progress').classList.add('hidden');
                alert(`Sync completed! Added: ${result.added}, Updated: ${result.updated}`);
            }, 2000);
        } else {
            const error = await response.text();
            showSyncError(`Failed to execute sync: ${error}`);
        }
    } catch (error) {
        showSyncError(`Error: ${error.message}`);
    } finally {
        button.disabled = false;
        button.innerText = originalText;
    }
}

function displaySyncResults(results) {
    document.getElementById('sync-new-count').innerText = results.new_servers?.length || 0;
    document.getElementById('sync-update-count').innerText = results.updates?.length || 0;
    document.getElementById('sync-unchanged-count').innerText = results.unchanged || 0;

    const detailsDiv = document.getElementById('sync-details');
    detailsDiv.innerHTML = '';

    // Show new servers
    if (results.new_servers && results.new_servers.length > 0) {
        detailsDiv.innerHTML += '<div class="font-medium text-green-700 mb-1">New Servers:</div>';
        results.new_servers.forEach(server => {
            detailsDiv.innerHTML += `
                <div class="ml-4 text-sm text-gray-700">
                    <span class="font-medium">${server.name}</span> - ${server.description}
                </div>`;
        });
    }

    // Show updates
    if (results.updates && results.updates.length > 0) {
        detailsDiv.innerHTML += '<div class="font-medium text-blue-700 mb-1 mt-2">Updates Available:</div>';
        results.updates.forEach(update => {
            detailsDiv.innerHTML += `
                <div class="ml-4 text-sm text-gray-700">
                    <span class="font-medium">${update.name}</span> -
                    ${update.current_version} → ${update.new_version}
                </div>`;
        });
    }

    document.getElementById('sync-results').classList.remove('hidden');
}

function showSyncError(message) {
    const errorDiv = document.getElementById('sync-error');
    errorDiv.querySelector('p').innerText = message;
    errorDiv.classList.remove('hidden');
}

function updateSyncProgress(percentage, message) {
    document.getElementById('sync-progress-bar').style.width = `${percentage}%`;
    document.getElementById('sync-progress-text').innerText = message;
}

async function validateImport() {
    const validationErrors = [];
    const selectedServers = getSelectedServers();
    
    for (const server of selectedServers) {
        // Check required fields
        if (!server.description) {
            validationErrors.push(`${server.key}: Missing description`);
        }
        if (!server.repository.url) {
            validationErrors.push(`${server.key}: Missing repository URL`);
        } else if (!server.repository.url.includes('github.com')) {
            validationErrors.push(`${server.key}: Repository must be a GitHub URL`);
        }
        
        // Validate against schema (if endpoint exists)
        try {
            const response = await apiRequest('/api/servers/validate', {
                method: 'POST',
                body: JSON.stringify(server)
            });
            
            if (response.ok) {
                const result = await response.json();
                if (!result.valid && result.errors) {
                    validationErrors.push(`${server.key}: ${result.errors.join(', ')}`);
                }
            }
        } catch (error) {
            // Validation endpoint might not exist yet
            console.log('Validation endpoint not available');
        }
    }
    
    const errorContainer = document.getElementById('import-validation-errors');
    const errorList = document.getElementById('import-errors-list');
    
    if (validationErrors.length > 0) {
        errorList.innerHTML = validationErrors.map(err => `<li>${err}</li>`).join('');
        errorContainer.classList.remove('hidden');
    } else {
        errorContainer.classList.add('hidden');
        alert('All servers validated successfully!');
    }
}

function getSelectedServers() {
    const selectedServers = [];
    
    importedServers.forEach((server, index) => {
        const checkbox = document.getElementById(`import-select-${index}`);
        if (checkbox && checkbox.checked) {
            // Update server with edited values
            server.name = document.getElementById(`import-name-${index}`).value;
            server.description = document.getElementById(`import-desc-${index}`).value;
            server.repository.url = document.getElementById(`import-repo-${index}`).value;
            server.version_detail.version = document.getElementById(`import-version-${index}`).value;
            
            // Generate repository ID from URL if possible
            if (server.repository.url && server.repository.url.includes('github.com')) {
                const match = server.repository.url.match(/github\.com\/([^\/]+)\/([^\/\.]+)/);
                if (match) {
                    server.repository.id = `github-${match[1]}-${match[2]}`;
                }
            }
            
            selectedServers.push(server);
        }
    });
    
    return selectedServers;
}

async function executeImport() {
    const selectedServers = getSelectedServers();
    
    if (selectedServers.length === 0) {
        alert('Please select at least one server to import');
        return;
    }
    
    // Validate required fields
    const invalid = selectedServers.filter(s => !s.description || !s.repository.url);
    if (invalid.length > 0) {
        alert('Please fill in all required fields (marked with *)');
        return;
    }
    
    try {
        const response = await apiRequest('/api/servers/import', {
            method: 'POST',
            body: JSON.stringify({
                servers: selectedServers,
                options: {
                    skip_existing: true,
                    validate_packages: false
                }
            })
        });
        
        if (response.ok) {
            const result = await response.json();
            displayImportResults(result);
        } else {
            // Fallback: import one by one if batch endpoint doesn't exist
            await importServersIndividually(selectedServers);
        }
        
    } catch (error) {
        alert(`Import failed: ${error.message}`);
    }
}

async function importServersIndividually(servers) {
    const results = {
        success: [],
        failed: []
    };
    
    for (const server of servers) {
        try {
            const response = await apiRequest('/api/servers', {
                method: 'POST',
                body: JSON.stringify(server)
            });
            
            if (response.ok) {
                const data = await response.json();
                results.success.push({ name: server.name, id: data.id });
            } else {
                const error = await response.text();
                results.failed.push({ name: server.name, error });
            }
        } catch (error) {
            results.failed.push({ name: server.name, error: error.message });
        }
    }
    
    displayImportResults({
        success: results.success,
        failed: results.failed,
        summary: {
            total: servers.length,
            success: results.success.length,
            failed: results.failed.length
        }
    });
}

function displayImportResults(results) {
    const summaryDiv = document.getElementById('import-results-summary');
    const detailsDiv = document.getElementById('import-results-details');
    
    summaryDiv.innerHTML = `
        <p class="text-lg">
            Total: ${results.summary?.total || results.success.length + results.failed.length} | 
            <span class="text-green-600">Success: ${results.summary?.success || results.success.length}</span> | 
            <span class="text-red-600">Failed: ${results.summary?.failed || results.failed.length}</span>
        </p>
    `;
    
    detailsDiv.innerHTML = '';
    
    if (results.success && results.success.length > 0) {
        const successDiv = document.createElement('div');
        successDiv.className = 'p-3 bg-green-100 rounded';
        successDiv.innerHTML = `
            <p class="font-semibold text-green-800 mb-1">Successfully Imported:</p>
            <ul class="text-sm text-green-700">
                ${results.success.map(s => `<li>✓ ${s.name}</li>`).join('')}
            </ul>
        `;
        detailsDiv.appendChild(successDiv);
    }
    
    if (results.failed && results.failed.length > 0) {
        const failedDiv = document.createElement('div');
        failedDiv.className = 'p-3 bg-red-100 rounded';
        failedDiv.innerHTML = `
            <p class="font-semibold text-red-800 mb-1">Failed:</p>
            <ul class="text-sm text-red-700">
                ${results.failed.map(f => `<li>✗ ${f.name}: ${f.error}</li>`).join('')}
            </ul>
        `;
        detailsDiv.appendChild(failedDiv);
    }
    
    // Show results step
    document.getElementById('import-step-2').classList.add('hidden');
    document.getElementById('import-step-3').classList.remove('hidden');
    
    // Reload servers table if any were imported successfully
    if (results.success && results.success.length > 0) {
        loadServers();
    }
}