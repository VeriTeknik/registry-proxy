// Check authentication on page load
document.addEventListener('DOMContentLoaded', () => {
    checkAuth();
    loadServers();
});

// Global variables
let currentServerId = null;
let serversTable = null;

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
        const response = await apiRequest('/api/servers?limit=100');
        const data = await response.json();
        
        // Update stats
        const servers = data.servers || [];
        document.getElementById('total-servers').textContent = servers.length;
        document.getElementById('active-servers').textContent = 
            servers.filter(s => s.status === 'active').length;
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
    const statusBadge = server.status === 'active' 
        ? '<span class="px-2 py-1 text-xs rounded-full bg-green-100 text-green-800">Active</span>'
        : '<span class="px-2 py-1 text-xs rounded-full bg-yellow-100 text-yellow-800">Deprecated</span>';
    
    const registryName = server.packages && server.packages[0] 
        ? server.packages[0].registry_name 
        : 'N/A';
    
    const actions = `
        <button onclick="editServer('${server.id}')" class="text-blue-600 hover:text-blue-900 mr-2">Edit</button>
        <button onclick="toggleStatus('${server.id}', '${server.status}')" class="text-yellow-600 hover:text-yellow-900 mr-2">Toggle</button>
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
    const newStatus = currentStatus === 'active' ? 'deprecated' : 'active';
    
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