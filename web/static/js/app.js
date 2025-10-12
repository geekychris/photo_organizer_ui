// Global JavaScript functionality for Photo Database

// Utility functions
window.formatBytes = function(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

// Format dates consistently
window.formatDate = function(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
};

// Toast notification system
window.showToast = function(message, type = 'info') {
    // Create toast container if it doesn't exist
    let toastContainer = document.getElementById('toast-container');
    if (!toastContainer) {
        toastContainer = document.createElement('div');
        toastContainer.id = 'toast-container';
        toastContainer.className = 'toast-container position-fixed top-0 end-0 p-3';
        toastContainer.style.zIndex = '9999';
        document.body.appendChild(toastContainer);
    }
    
    // Create toast
    const toast = document.createElement('div');
    toast.className = `toast align-items-center text-white bg-${type} border-0`;
    toast.setAttribute('role', 'alert');
    
    const toastId = 'toast-' + Date.now();
    toast.innerHTML = `
        <div class="d-flex">
            <div class="toast-body">
                ${message}
            </div>
            <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
        </div>
    `;
    
    toastContainer.appendChild(toast);
    
    const bsToast = new bootstrap.Toast(toast, { delay: 5000 });
    bsToast.show();
    
    // Remove toast element after it's hidden
    toast.addEventListener('hidden.bs.toast', () => {
        toast.remove();
    });
};

// Loading state management
window.setLoading = function(element, loading) {
    if (typeof element === 'string') {
        element = document.getElementById(element);
    }
    
    if (!element) return;
    
    if (loading) {
        element.disabled = true;
        element.dataset.originalText = element.innerHTML;
        element.innerHTML = '<span class="spinner-border spinner-border-sm me-2" role="status"></span>Loading...';
    } else {
        element.disabled = false;
        element.innerHTML = element.dataset.originalText;
    }
};

// API helper functions
window.apiRequest = async function(url, options = {}) {
    const config = {
        headers: {
            'Content-Type': 'application/json',
            ...options.headers
        },
        ...options
    };
    
    try {
        const response = await fetch(url, config);
        
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`HTTP ${response.status}: ${errorText}`);
        }
        
        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
            return await response.json();
        }
        
        return await response.text();
    } catch (error) {
        console.error('API request failed:', error);
        throw error;
    }
};

// Image loading with fallback
window.loadImageWithFallback = function(img, primarySrc, fallbackSrc) {
    img.onerror = function() {
        if (fallbackSrc && this.src !== fallbackSrc) {
            this.src = fallbackSrc;
        } else {
            // Show placeholder
            this.style.backgroundColor = '#f8f9fa';
            this.style.display = 'flex';
            this.style.alignItems = 'center';
            this.style.justifyContent = 'center';
            this.innerHTML = '<i class="bi bi-image text-muted"></i>';
        }
    };
    
    img.src = primarySrc;
};

// Infinite scroll helper
window.createInfiniteScroll = function(callback, threshold = 1000) {
    let loading = false;
    
    const scrollHandler = function() {
        if (loading) return;
        
        if (window.innerHeight + window.scrollY >= document.body.offsetHeight - threshold) {
            loading = true;
            callback().finally(() => {
                loading = false;
            });
        }
    };
    
    window.addEventListener('scroll', scrollHandler);
    
    return function cleanup() {
        window.removeEventListener('scroll', scrollHandler);
    };
};

// Debounce function for search inputs
window.debounce = function(func, wait, immediate) {
    let timeout;
    return function executedFunction(...args) {
        const later = function() {
            timeout = null;
            if (!immediate) func.apply(this, args);
        };
        
        const callNow = immediate && !timeout;
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
        
        if (callNow) func.apply(this, args);
    };
};

// Local storage helpers
window.storage = {
    set(key, value) {
        try {
            localStorage.setItem(`photodb_${key}`, JSON.stringify(value));
        } catch (e) {
            console.warn('Failed to save to localStorage:', e);
        }
    },
    
    get(key, defaultValue = null) {
        try {
            const item = localStorage.getItem(`photodb_${key}`);
            return item ? JSON.parse(item) : defaultValue;
        } catch (e) {
            console.warn('Failed to read from localStorage:', e);
            return defaultValue;
        }
    },
    
    remove(key) {
        try {
            localStorage.removeItem(`photodb_${key}`);
        } catch (e) {
            console.warn('Failed to remove from localStorage:', e);
        }
    }
};

// Initialize global functionality
document.addEventListener('DOMContentLoaded', function() {
    // Set active navigation item
    const currentPath = window.location.pathname;
    const navLinks = document.querySelectorAll('.nav-link');
    
    navLinks.forEach(link => {
        if (link.getAttribute('href') === currentPath) {
            link.classList.add('active');
        } else {
            link.classList.remove('active');
        }
    });
    
    // Initialize tooltips
    const tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
    tooltipTriggerList.map(function(tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl);
    });
    
    // Initialize popovers
    const popoverTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="popover"]'));
    popoverTriggerList.map(function(popoverTriggerEl) {
        return new bootstrap.Popover(popoverTriggerEl);
    });
    
    // Add keyboard shortcuts
    document.addEventListener('keydown', function(e) {
        // Ctrl/Cmd + K for search
        if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
            e.preventDefault();
            const searchInput = document.querySelector('input[type="search"], input[placeholder*="search"], input[placeholder*="Search"]');
            if (searchInput) {
                searchInput.focus();
            }
        }
        
        // Escape to close modals
        if (e.key === 'Escape') {
            const openModals = document.querySelectorAll('.modal.show');
            openModals.forEach(modal => {
                bootstrap.Modal.getInstance(modal)?.hide();
            });
        }
    });
});

// Error handling for images
document.addEventListener('DOMContentLoaded', function() {
    // Handle broken thumbnail images
    document.addEventListener('error', function(e) {
        if (e.target.tagName === 'IMG' || e.target.style.backgroundImage) {
            // Replace with placeholder
            if (e.target.tagName === 'IMG') {
                e.target.style.display = 'none';
                const placeholder = document.createElement('div');
                placeholder.className = 'photo-thumbnail-placeholder d-flex align-items-center justify-content-center bg-light';
                placeholder.style.width = e.target.width || '100px';
                placeholder.style.height = e.target.height || '100px';
                placeholder.innerHTML = '<i class="bi bi-image text-muted"></i>';
                e.target.parentNode.insertBefore(placeholder, e.target.nextSibling);
            }
        }
    }, true);
});

// Export for use in other scripts
window.PhotoDB = {
    formatBytes,
    formatDate,
    showToast,
    setLoading,
    apiRequest,
    loadImageWithFallback,
    createInfiniteScroll,
    debounce,
    storage
};