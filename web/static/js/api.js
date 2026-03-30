// VaultDrift API Client

const API_BASE = '';

class VaultDriftAPI {
    constructor() {
        this.token = localStorage.getItem('token');
    }

    setToken(token) {
        this.token = token;
        localStorage.setItem('token', token);
    }

    clearToken() {
        this.token = null;
        localStorage.removeItem('token');
    }

    async request(method, path, body = null, query = null) {
        const url = new URL(API_BASE + path, window.location.origin);

        if (query) {
            Object.keys(query).forEach(key => {
                if (query[key] !== undefined && query[key] !== null) {
                    url.searchParams.append(key, query[key]);
                }
            });
        }

        const options = {
            method,
            headers: {
                'Content-Type': 'application/json'
            }
        };

        if (this.token) {
            options.headers['Authorization'] = `Bearer ${this.token}`;
        }

        if (body) {
            options.body = JSON.stringify(body);
        }

        const response = await fetch(url, options);

        if (response.status === 401) {
            this.clearToken();
            window.location.reload();
            return null;
        }

        if (!response.ok) {
            const error = await response.text();
            throw new Error(error || `HTTP ${response.status}`);
        }

        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
            return response.json();
        }

        return response;
    }

    // Auth
    async login(username, password) {
        const response = await this.request('POST', '/api/v1/auth/login', {
            username,
            password
        });

        if (response && response.data) {
            this.setToken(response.data.token);
            return response.data;
        }

        throw new Error('Invalid response');
    }

    async logout() {
        try {
            await this.request('POST', '/api/v1/auth/logout');
        } finally {
            this.clearToken();
        }
    }

    // Files
    async listFiles(parentId = '', limit = 100, offset = 0) {
        const query = { limit, offset };
        if (parentId) query.parent_id = parentId;

        const response = await this.request('GET', '/api/v1/files', null, query);
        return response?.data?.files || [];
    }

    async getFile(fileId) {
        const response = await this.request('GET', `/api/v1/files/${fileId}`);
        return response?.data;
    }

    async createFile(parentId, name, mimeType, size) {
        const body = { parent_id: parentId, name, mime_type: mimeType, size };
        const response = await this.request('POST', '/api/v1/files', body);
        return response?.data;
    }

    async renameFile(fileId, newName) {
        const response = await this.request('PUT', `/api/v1/files/${fileId}`, { name: newName });
        return response?.data;
    }

    async moveFile(fileId, newParentId) {
        const response = await this.request('PUT', `/api/v1/files/${fileId}`, { parent_id: newParentId });
        return response?.data;
    }

    async deleteFile(fileId) {
        return this.request('DELETE', `/api/v1/files/${fileId}`);
    }

    // Folders
    async createFolder(name, parentId = '') {
        const body = { name };
        if (parentId) body.parent_id = parentId;

        const response = await this.request('POST', '/api/v1/folders', body);
        return response?.data;
    }

    async deleteFolder(folderId) {
        return this.request('DELETE', `/api/v1/folders/${folderId}`);
    }

    async getBreadcrumbs(folderId) {
        const response = await this.request('GET', `/api/v1/folders/${folderId}/breadcrumbs`);
        return response?.data?.breadcrumbs || [];
    }

    // Upload/Download
    async getUploadUrl(parentId, name, mimeType, size) {
        const body = { parent_id: parentId, name, mime_type: mimeType, size };
        const response = await this.request('POST', '/api/v1/uploads', body);
        return response?.data;
    }

    async uploadFile(file, parentId = '', onProgress = null) {
        // Get upload URL
        const uploadInfo = await this.getUploadUrl(parentId, file.name, file.type, file.size);

        // Upload to presigned URL
        const uploadResponse = await fetch(uploadInfo.upload_url, {
            method: 'PUT',
            body: file,
            headers: {
                'Content-Type': file.type
            }
        });

        if (!uploadResponse.ok) {
            throw new Error('Upload failed');
        }

        return uploadInfo.file_id;
    }

    async getDownloadUrl(fileId) {
        const response = await this.request('GET', `/api/v1/downloads/${fileId}`);
        return response?.data;
    }

    async downloadFile(fileId, fileName) {
        const downloadInfo = await this.getDownloadUrl(fileId);

        const response = await fetch(downloadInfo.download_url);
        if (!response.ok) {
            throw new Error('Download failed');
        }

        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);

        const a = document.createElement('a');
        a.href = url;
        a.download = fileName || downloadInfo.filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);

        window.URL.revokeObjectURL(url);
    }

    // Search
    async search(query, limit = 50) {
        const response = await this.request('GET', '/api/v1/files/search', null, { q: query, limit });
        return response?.data?.files || [];
    }

    // Shares
    async createShare(fileId, options = {}) {
        const body = {
            share_type: 'link',
            permission: 'read',
            ...options
        };

        const response = await this.request('POST', `/api/v1/files/${fileId}/shares`, body);
        return response?.data;
    }

    async listShares(fileId) {
        const response = await this.request('GET', `/api/v1/files/${fileId}/shares`);
        return response?.data?.shares || [];
    }

    async revokeShare(shareId) {
        return this.request('DELETE', `/api/v1/shares/${shareId}`);
    }

    // Health
    async checkHealth() {
        try {
            const response = await fetch(`${API_BASE}/health`);
            return response.ok;
        } catch {
            return false;
        }
    }
}

// Create global API instance
const api = new VaultDriftAPI();
