class ConfigManager {
    constructor() {
        this.init();
    }

    async init() {
        try {
            await this.loadFeedConfiguration();
        } catch (error) {
            console.error('Failed to initialize config manager:', error);
            this.showError('Failed to load configuration: ' + error.message);
        }
    }

    async loadFeedConfiguration() {
        try {
            const response = await fetch('/api/v1/feeds');
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            const data = await response.json();
            this.renderFeedConfiguration(data.feeds);
        } catch (error) {
            console.error('Failed to load feed configuration:', error);
            this.showError('Failed to load feed configuration: ' + error.message);
        }
    }

    renderFeedConfiguration(feeds) {
        const configContainer = document.getElementById('feedConfig');
        if (!configContainer) return;

        if (!feeds || Object.keys(feeds).length === 0) {
            configContainer.innerHTML = `
                <div class="no-config">
                    <i class="fas fa-exclamation-circle"></i>
                    <p>No feed configuration found</p>
                </div>
            `;
            return;
        }

        const configHTML = Object.entries(feeds).map(([topic, config]) => {
            const urls = config.urls || [];
            const filters = config.filters || [];
            const feeds = config.feeds || [];
            
            return `
                <div class="config-card">
                    <div class="config-header">
                        <h3><i class="fas fa-tag"></i> ${this.escapeHtml(topic)}</h3>
                        <span class="topic-badge">${topic}</span>
                    </div>
                    
                    <div class="config-content">
                        <div class="config-section">
                            <h4><i class="fas fa-link"></i> Feed URLs (${urls.length})</h4>
                            ${urls.length > 0 ? `
                                <ul class="url-list">
                                    ${urls.map(url => {
                                        const feed = feeds.find(f => f.url === url);
                                        const healthIcon = this.getHealthIcon(feed);
                                        const healthTooltip = feed ? this.getHealthTooltip(feed) : '';
                                        
                                        return `
                                            <li class="url-item">
                                                <div class="url-header">
                                                    ${healthIcon}
                                                    <a href="${this.escapeHtml(url)}" target="_blank" class="feed-url">
                                                        ${this.escapeHtml(url)}
                                                    </a>
                                                </div>
                                                ${healthTooltip ? `<div class="health-tooltip">${healthTooltip}</div>` : ''}
                                            </li>
                                        `;
                                    }).join('')}
                                </ul>
                            ` : `
                                <p class="no-urls">No feed URLs configured</p>
                            `}
                        </div>
                        
                        <div class="config-section">
                            <h4><i class="fas fa-filter"></i> Filters (${filters.length})</h4>
                            ${filters.length > 0 ? `
                                <ul class="filter-list">
                                    ${filters.map(filter => `
                                        <li class="filter-item">
                                            <i class="fas fa-tag"></i>
                                            <span class="filter-tag">${this.escapeHtml(filter)}</span>
                                        </li>
                                    `).join('')}
                                </ul>
                            ` : `
                                <p class="no-filters">No filters configured</p>
                            `}
                        </div>
                    </div>
                </div>
            `;
        }).join('');

        configContainer.innerHTML = configHTML;
    }

    renderFeedStatus(feeds) {
        const statusContainer = document.getElementById('feedStatus');
        if (!statusContainer) return;

        if (!feeds || Object.keys(feeds).length === 0) {
            statusContainer.innerHTML = `
                <div class="no-status">
                    <i class="fas fa-exclamation-circle"></i>
                    <p>No feed status available</p>
                </div>
            `;
            return;
        }

        let disabledFeeds = [];
        let activeFeeds = [];
        let errorFeeds = [];
        let retryFeeds = [];

        Object.entries(feeds).forEach(([topic, config]) => {
            const feeds = config.feeds || [];
            feeds.forEach(feed => {
                if (feed.is_disabled) {
                    if (feed.is_content_issue) {
                        disabledFeeds.push({ ...feed, topic });
                    } else {
                        retryFeeds.push({ ...feed, topic });
                    }
                } else if (feed.last_error) {
                    errorFeeds.push({ ...feed, topic });
                } else {
                    activeFeeds.push({ ...feed, topic });
                }
            });
        });

        const statusHTML = `
            <div class="status-summary">
                <div class="status-card active">
                    <h4><i class="fas fa-check-circle"></i> Active Feeds (${activeFeeds.length})</h4>
                    ${activeFeeds.length > 0 ? `
                        <ul class="feed-status-list">
                            ${activeFeeds.map(feed => `
                                <li class="feed-status-item active">
                                    <div class="feed-status-header">
                                        <span class="feed-topic">${this.escapeHtml(feed.topic)}</span>
                                        <span class="feed-articles">${feed.articles_count} articles</span>
                                    </div>
                                    <div class="feed-url">${this.escapeHtml(feed.url)}</div>
                                    <div class="feed-last-polled">Last polled: ${feed.last_polled ? new Date(feed.last_polled).toLocaleString() : 'Never'}</div>
                                    ${feed.last_success ? `<div class="feed-last-success">Last success: ${new Date(feed.last_success).toLocaleString()}</div>` : ''}
                                    ${feed.user_agent ? `<div class="feed-user-agent">User-Agent: ${this.escapeHtml(feed.user_agent.substring(0, 50))}${feed.user_agent.length > 50 ? '...' : ''}</div>` : ''}
                                </li>
                            `).join('')}
                        </ul>
                    ` : '<p>No active feeds</p>'}
                </div>

                <div class="status-card error">
                    <h4><i class="fas fa-exclamation-triangle"></i> Feeds with Errors (${errorFeeds.length})</h4>
                    ${errorFeeds.length > 0 ? `
                        <ul class="feed-status-list">
                            ${errorFeeds.map(feed => `
                                <li class="feed-status-item error">
                                    <div class="feed-status-header">
                                        <span class="feed-topic">${this.escapeHtml(feed.topic)}</span>
                                        <span class="feed-error-count">${feed.consecutive_errors} consecutive errors</span>
                                    </div>
                                    <div class="feed-url">${this.escapeHtml(feed.url)}</div>
                                    <div class="feed-error">${this.escapeHtml(feed.last_error)}</div>
                                    <div class="feed-last-polled">Last polled: ${feed.last_polled ? new Date(feed.last_polled).toLocaleString() : 'Never'}</div>
                                    ${feed.next_retry ? `<div class="feed-next-retry">Next retry: ${new Date(feed.next_retry).toLocaleString()}</div>` : ''}
                                    ${feed.user_agent ? `<div class="feed-user-agent">User-Agent: ${this.escapeHtml(feed.user_agent.substring(0, 50))}${feed.user_agent.length > 50 ? '...' : ''}</div>` : ''}
                                    ${feed.tested_user_agents && feed.tested_user_agents.length > 0 ? `<div class="feed-tested-uas">Tested ${feed.tested_user_agents.length} User-Agents</div>` : ''}
                                </li>
                            `).join('')}
                        </ul>
                    ` : '<p>No feeds with errors</p>'}
                </div>

                <div class="status-card retry">
                    <h4><i class="fas fa-clock"></i> Feeds in Retry Mode (${retryFeeds.length})</h4>
                    ${retryFeeds.length > 0 ? `
                        <ul class="feed-status-list">
                            ${retryFeeds.map(feed => `
                                <li class="feed-status-item retry">
                                    <div class="feed-status-header">
                                        <span class="feed-topic">${this.escapeHtml(feed.topic)}</span>
                                        <span class="feed-retry-count">${feed.retry_count} retries</span>
                                    </div>
                                    <div class="feed-url">${this.escapeHtml(feed.url)}</div>
                                    <div class="feed-disabled-reason">${this.escapeHtml(feed.disabled_reason)}</div>
                                    <div class="feed-last-polled">Last polled: ${feed.last_polled ? new Date(feed.last_polled).toLocaleString() : 'Never'}</div>
                                    ${feed.next_retry ? `<div class="feed-next-retry">Next retry: ${new Date(feed.next_retry).toLocaleString()}</div>` : ''}
                                </li>
                            `).join('')}
                        </ul>
                    ` : '<p>No feeds in retry mode</p>'}
                </div>

                <div class="status-card disabled">
                    <h4><i class="fas fa-ban"></i> Permanently Disabled Feeds (${disabledFeeds.length})</h4>
                    ${disabledFeeds.length > 0 ? `
                        <ul class="feed-status-list">
                            ${disabledFeeds.map(feed => `
                                <li class="feed-status-item disabled">
                                    <div class="feed-status-header">
                                        <span class="feed-topic">${this.escapeHtml(feed.topic)}</span>
                                        <span class="feed-content-issue">Content Issue</span>
                                    </div>
                                    <div class="feed-url">${this.escapeHtml(feed.url)}</div>
                                    <div class="feed-disabled-reason">${this.escapeHtml(feed.disabled_reason)}</div>
                                    <div class="feed-last-polled">Last polled: ${feed.last_polled ? new Date(feed.last_polled).toLocaleString() : 'Never'}</div>
                                </li>
                            `).join('')}
                        </ul>
                    ` : '<p>No permanently disabled feeds</p>'}
                </div>
            </div>
        `;

        statusContainer.innerHTML = statusHTML;
    }

    showError(message) {
        const errorDiv = document.createElement('div');
        errorDiv.className = 'error-message';
        errorDiv.innerHTML = `
            <i class="fas fa-exclamation-triangle"></i>
            <span>${this.escapeHtml(message)}</span>
        `;
        
        const mainContent = document.querySelector('.main-content');
        if (mainContent) {
            mainContent.insertBefore(errorDiv, mainContent.firstChild);
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    getHealthIcon(feed) {
        if (!feed) {
            return '<i class="fas fa-question-circle health-icon unknown" title="Not yet polled"></i>';
        }
        
        switch (feed.status) {
            case 'healthy':
                return '<i class="fas fa-check-circle health-icon healthy" title="Healthy"></i>';
            case 'warning':
                return '<i class="fas fa-exclamation-triangle health-icon warning" title="Warning"></i>';
            case 'error':
                return '<i class="fas fa-times-circle health-icon error" title="Error"></i>';
            case 'disabled':
                return '<i class="fas fa-ban health-icon disabled" title="Disabled"></i>';
            case 'unknown':
            default:
                return '<i class="fas fa-question-circle health-icon unknown" title="Unknown status"></i>';
        }
    }

    getHealthTooltip(feed) {
        if (!feed || feed.status === 'healthy') return '';
        
        let tooltip = `<strong>Status:</strong> ${feed.status}`;
        if (feed.reason) {
            tooltip += `<br><strong>Reason:</strong> ${this.escapeHtml(feed.reason)}`;
        }
        if (feed.articles_count !== undefined) {
            tooltip += `<br><strong>Articles:</strong> ${feed.articles_count}`;
        }
        if (feed.last_polled) {
            tooltip += `<br><strong>Last polled:</strong> ${feed.last_polled}`;
        }
        
        return tooltip;
    }
}

// Initialize the config manager when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        new ConfigManager();
    });
} else {
    new ConfigManager();
} 