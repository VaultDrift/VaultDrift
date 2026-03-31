// VaultDrift Web App

document.addEventListener('DOMContentLoaded', () => {
    // State
    let currentFolder = '';
    let files = [];
    let currentView = 'files';
    let selectedFile = null;

    // Elements
    const loginScreen = document.getElementById('login-screen');
    const mainScreen = document.getElementById('main-screen');
    const loginForm = document.getElementById('login-form');
    const loginError = document.getElementById('login-error');
    const fileListBody = document.getElementById('file-list-body');
    const breadcrumbs = document.getElementById('breadcrumbs');
    const emptyState = document.getElementById('empty-state');
    const fileList = document.getElementById('file-list');
    const logoutBtn = document.getElementById('logout-btn');
    const newFolderBtn = document.getElementById('new-folder-btn');
    const uploadBtn = document.getElementById('upload-btn');
    const fileInput = document.getElementById('file-input');
    const searchInput = document.getElementById('search-input');
    const navItems = document.querySelectorAll('.nav-item');

    // Modals
    const newFolderModal = document.getElementById('new-folder-modal');
    const shareModal = document.getElementById('share-modal');
    const createFolderConfirm = document.getElementById('create-folder-confirm');
    const createShareConfirm = document.getElementById('create-share-confirm');

    // Check auth status
    if (api.token) {
        showMainScreen();
    } else {
        showLoginScreen();
    }

    // Login
    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        loginError.textContent = '';

        const username = document.getElementById('username').value;
        const password = document.getElementById('password').value;

        try {
            await api.login(username, password);
            showMainScreen();
        } catch (err) {
            loginError.textContent = err.message || 'Login failed';
        }
    });

    // Logout
    logoutBtn.addEventListener('click', async () => {
        await api.logout();
        showLoginScreen();
    });

    // Navigation
    navItems.forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            const view = item.dataset.view;

            navItems.forEach(n => n.classList.remove('active'));
            item.classList.add('active');

            currentView = view;
            currentFolder = '';

            switch (view) {
                case 'files':
                    loadFiles();
                    break;
                case 'shared':
                    loadShared();
                    break;
                case 'recent':
                    loadRecent();
                    break;
                case 'trash':
                    loadTrash();
                    break;
            }
        });
    });

    // New Folder
    newFolderBtn.addEventListener('click', () => {
        showModal(newFolderModal);
        document.getElementById('new-folder-name').focus();
    });

    createFolderConfirm.addEventListener('click', async () => {
        const name = document.getElementById('new-folder-name').value.trim();
        if (!name) return;

        try {
            await api.createFolder(name, currentFolder);
            hideModal(newFolderModal);
            document.getElementById('new-folder-name').value = '';
            loadFiles();
            showToast('Folder created', 'success');
        } catch (err) {
            showToast(err.message, 'error');
        }
    });

    // Upload
    uploadBtn.addEventListener('click', () => {
        fileInput.click();
    });

    document.getElementById('empty-upload-btn')?.addEventListener('click', () => {
        fileInput.click();
    });

    fileInput.addEventListener('change', async () => {
        const files = fileInput.files;
        if (!files.length) return;

        for (const file of files) {
            try {
                await api.uploadFile(file, currentFolder);
                showToast(`Uploaded ${file.name}`, 'success');
            } catch (err) {
                showToast(`Failed to upload ${file.name}: ${err.message}`, 'error');
            }
        }

        fileInput.value = '';
        loadFiles();
    });

    // Search
    let searchTimeout;
    searchInput.addEventListener('input', () => {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(async () => {
            const query = searchInput.value.trim();
            if (query) {
                const results = await api.search(query);
                renderFiles(results);
            } else {
                loadFiles();
            }
        }, 300);
    });

    // Modal close buttons
    document.querySelectorAll('[data-close-modal]').forEach(btn => {
        btn.addEventListener('click', () => {
            hideModal(newFolderModal);
            hideModal(shareModal);
        });
    });

    // Close modals on outside click
    [newFolderModal, shareModal].forEach(modal => {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                hideModal(modal);
            }
        });
    });

    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            hideModal(newFolderModal);
            hideModal(shareModal);
        }
    });

    // Functions
    function showLoginScreen() {
        loginScreen.classList.add('active');
        mainScreen.classList.remove('active');
    }

    function showMainScreen() {
        loginScreen.classList.remove('active');
        mainScreen.classList.add('active');
        loadFiles();
    }

    async function loadFiles() {
        try {
            files = await api.listFiles(currentFolder);
            renderFiles(files);
            updateBreadcrumbs();
        } catch (err) {
            showToast('Failed to load files', 'error');
        }
    }

    async function loadShared() {
        renderFiles([]);
        updateBreadcrumbs([{ name: 'Shared with me' }]);
    }

    async function loadRecent() {
        renderFiles([]);
        updateBreadcrumbs([{ name: 'Recent files' }]);
    }

    async function loadTrash() {
        renderFiles([]);
        updateBreadcrumbs([{ name: 'Trash' }]);
    }

    function renderFiles(files) {
        if (!files || files.length === 0) {
            fileList.classList.add('hidden');
            emptyState.classList.remove('hidden');
            return;
        }

        fileList.classList.remove('hidden');
        emptyState.classList.add('hidden');

        fileListBody.innerHTML = files.map(file => `
            <tr data-file-id="${file.id}" data-file-type="${file.type}">
                <td>
                    <div class="file-name" onclick="navigateTo('${file.id}', '${file.type}')">
                        <span class="file-icon">
                            ${getFileIcon(file.type, file.mime_type)}
                        </span>
                        ${escapeHtml(file.name)}
                    </div>
                </td>
                <td>${file.type === 'folder' ? '--' : formatBytes(file.size_bytes || 0)}</td>
                <td>${formatDate(file.updated_at)}</td>
                <td class="col-actions">
                    <button class="btn btn-sm btn-secondary" onclick="showFileActions('${file.id}', event)">
                        ⋮
                    </button>
                </td>
            </tr>
        `).join('');
    }

    function updateBreadcrumbs(customPath) {
        if (customPath) {
            breadcrumbs.innerHTML = customPath.map((item, i) => `
                <span class="breadcrumb-item">${escapeHtml(item.name)}</span>
            `).join(' <span>/</span> ');
            return;
        }

        if (!currentFolder) {
            breadcrumbs.innerHTML = '<span class="breadcrumb-item">My Files</span>';
            return;
        }

        // TODO: Load actual breadcrumbs from API
        breadcrumbs.innerHTML = `
            <span class="breadcrumb-item" onclick="navigateToRoot()">My Files</span>
            <span>/</span>
            <span class="breadcrumb-item">Current Folder</span>
        `;
    }

    function getFileIcon(type, mimeType) {
        if (type === 'folder') {
            return `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            </svg>`;
        }

        // File icons based on mime type
        if (mimeType?.startsWith('image/')) {
            return `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="18" height="18" rx="2"/>
                <circle cx="8.5" cy="8.5" r="1.5"/>
                <path d="M21 15l-5-5L5 21"/>
            </svg>`;
        }

        if (mimeType?.startsWith('video/')) {
            return `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="2" y="2" width="20" height="20" rx="2"/>
                <polygon points="10 8 16 12 10 16"/>
            </svg>`;
        }

        return `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
            <polyline points="14 2 14 8 20 8"/>
        </svg>`;
    }

    function formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    function formatDate(timestamp) {
        if (!timestamp) return '--';
        // API returns ISO string format, not unix timestamp
        const date = new Date(timestamp);
        if (isNaN(date.getTime())) return '--';
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }

    function escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    function showModal(modal) {
        modal.classList.add('active');
    }

    function hideModal(modal) {
        modal.classList.remove('active');
    }

    function showToast(message, type = 'info') {
        let container = document.querySelector('.toast-container');
        if (!container) {
            container = document.createElement('div');
            container.className = 'toast-container';
            document.body.appendChild(container);
        }

        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;

        container.appendChild(toast);

        setTimeout(() => {
            toast.remove();
        }, 3000);
    }

    // Global functions for inline handlers
    window.navigateTo = (id, type) => {
        if (type === 'folder') {
            currentFolder = id;
            loadFiles();
        } else {
            // Show file preview or download
            api.downloadFile(id, files.find(f => f.id === id)?.name);
        }
    };

    window.navigateToRoot = () => {
        currentFolder = '';
        loadFiles();
    };

    window.showFileActions = (fileId, event) => {
        const file = files.find(f => f.id === fileId);
        if (!file) return;

        // Remove any existing menus
        document.querySelectorAll('.context-menu').forEach(m => m.remove());

        // Create context menu
        const menu = document.createElement('div');
        menu.className = 'context-menu';
        menu.style.position = 'absolute';
        menu.style.zIndex = '1000';
        menu.innerHTML = `
            <div class="context-menu-item" onclick="renameFile('${fileId}')">Rename</div>
            <div class="context-menu-item" onclick="moveFile('${fileId}')">Move</div>
            <div class="context-menu-item" onclick="shareFile('${fileId}')">Share</div>
            <div class="context-menu-item" onclick="downloadFile('${fileId}')">Download</div>
            <div class="context-menu-item danger" onclick="deleteFile('${fileId}')">Delete</div>
        `;

        // Position menu near the clicked button
        const btn = event?.target || event?.currentTarget;
        if (btn) {
            const rect = btn.getBoundingClientRect();
            menu.style.top = (rect.bottom + window.scrollY) + 'px';
            menu.style.left = (rect.left + window.scrollX - 120) + 'px';
        }

        document.body.appendChild(menu);

        const closeMenu = (e) => {
            if (!menu.contains(e.target)) {
                menu.remove();
                document.removeEventListener('click', closeMenu);
            }
        };

        setTimeout(() => {
            document.addEventListener('click', closeMenu);
        }, 0);
    };

    window.renameFile = async (fileId) => {
        const file = files.find(f => f.id === fileId);
        if (!file) return;

        const newName = prompt('New name:', file.name);
        if (!newName || newName === file.name) return;

        try {
            await api.renameFile(fileId, newName);
            loadFiles();
            showToast('File renamed', 'success');
        } catch (err) {
            showToast(err.message, 'error');
        }
    };

    window.moveFile = async (fileId) => {
        // TODO: Show folder picker
        showToast('Move feature coming soon', 'info');
    };

    window.shareFile = async (fileId) => {
        selectedFile = fileId;
        document.getElementById('share-link-container').classList.add('hidden');
        showModal(shareModal);
    };

    window.downloadFile = async (fileId) => {
        const file = files.find(f => f.id === fileId);
        if (!file) return;

        try {
            await api.downloadFile(fileId, file.name);
            showToast('Download started', 'success');
        } catch (err) {
            showToast(err.message, 'error');
        }
    };

    window.deleteFile = async (fileId) => {
        const file = files.find(f => f.id === fileId);
        if (!file) return;

        if (!confirm(`Delete "${file.name}"?`)) return;

        try {
            if (file.type === 'folder') {
                await api.deleteFolder(fileId);
            } else {
                await api.deleteFile(fileId);
            }
            loadFiles();
            showToast('Deleted', 'success');
        } catch (err) {
            showToast(err.message, 'error');
        }
    };

    // Share modal handlers
    document.getElementById('share-expires').addEventListener('change', (e) => {
        document.getElementById('share-expires-days').disabled = !e.target.checked;
    });

    document.getElementById('share-password').addEventListener('change', (e) => {
        document.getElementById('share-password-value').disabled = !e.target.checked;
    });

    createShareConfirm.addEventListener('click', async () => {
        if (!selectedFile) return;

        const options = {
            permission: 'read',
            share_type: 'link'
        };

        if (document.getElementById('share-expires').checked) {
            options.expires_days = parseInt(document.getElementById('share-expires-days').value);
        }

        if (document.getElementById('share-password').checked) {
            options.password = document.getElementById('share-password-value').value;
        }

        try {
            const result = await api.createShare(selectedFile, options);

            if (result.share_url) {
                document.getElementById('share-link').value = result.share_url;
                document.getElementById('share-link-container').classList.remove('hidden');
            }

            showToast('Share link created', 'success');
        } catch (err) {
            showToast(err.message, 'error');
        }
    });

    document.getElementById('copy-share-link').addEventListener('click', () => {
        const input = document.getElementById('share-link');
        input.select();
        document.execCommand('copy');
        showToast('Link copied to clipboard', 'success');
    });
});
