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

    // Fetch and display feed configuration and statistics
    async loadFeedConfiguration() {
        try {
            // Fetch feed configuration and status
            const feedsResponse = await fetch('/api/v1/feeds');
            const feedsData = await feedsResponse.json();
            
            // Fetch feed statistics
            const statsResponse = await fetch('/api/v1/feeds/stats');
            const statsData = await statsResponse.json();
            
            // Create a map of feed statistics by source
            const feedStats = {};
            if (statsData.feeds) {
                statsData.feeds.forEach(feed => {
                    feedStats[feed.source] = feed;
                });
            }
            
            this.renderFeedConfiguration(feedsData.feeds, feedStats);
        } catch (error) {
            console.error('Error loading feed configuration:', error);
            document.getElementById('feedConfig').innerHTML = '<p class="error">Error loading feed configuration</p>';
        }
    }

    // Helper function to extract source name from URL
    getSourceFromUrl(url) {
        try {
            const urlObj = new URL(url);
            const hostname = urlObj.hostname;
            
            // Map common hostnames to readable source names
            const sourceMap = {
                'feeds.feedburner.com': 'Feedburner',
                'feeds.reuters.com': 'Reuters',
                'rss.cnn.com': 'CNN',
                'feeds.bbci.co.uk': 'BBC',
                'feeds.npr.org': 'NPR',
                'blog.golang.org': 'The Go Blog',
                'www.reddit.com': 'Reddit',
                'feeds.arstechnica.com': 'Ars Technica',
                'feeds.feedburner.com': 'Feedburner'
            };
            
            return sourceMap[hostname] || hostname;
        } catch (e) {
            return 'Unknown Source';
        }
    }

    renderFeedConfiguration(topics, feedStats) {
        const configContainer = document.getElementById('feedConfig');
        if (!configContainer) return;

        let html = '<div class="feed-config-grid">';
        
        Object.entries(topics).forEach(([topicName, topicData]) => {
            html += `
                <div class="topic-card">
                    <div class="topic-header">
                        <h2>${topicName.toUpperCase()}</h2>
                    </div>
                    <div class="topic-filters">
                        <h4>Filters:</h4>
                        <div class="filter-tags">
                            ${topicData.filters ? topicData.filters.map(filter => 
                                `<span class="filter-tag">${this.escapeHtml(filter)}</span>`
                            ).join('') : '<span class="no-filters">No filters</span>'}
                        </div>
                    </div>
                    <div class="topic-feeds">
                        <h4>Feeds:</h4>
                        ${topicData.feeds ? topicData.feeds.map(feed => {
                            const healthIcon = this.getHealthIcon(feed);
                            const sourceName = this.getSourceFromUrl(feed.url);
                            const stats = feedStats[sourceName] || {};
                            
                            return `
                                <div class="feed-item">
                                    <div class="feed-header">
                                        <span class="feed-source">${sourceName}</span>
                                        <span class="health-icon" title="${feed.last_error || feed.reason || 'Healthy'}">${healthIcon}</span>
                                    </div>
                                    <div class="feed-details">
                                        <p><strong>URL:</strong> <a href="${feed.url}" target="_blank">${feed.url}</a></p>
                                        <p><strong>Status:</strong> <span class="status-${feed.status}">${feed.status}</span></p>
                                        <p><strong>Articles:</strong> ${feed.articles_count || 0}</p>
                                        <p><strong>Last Polled:</strong> ${new Date(feed.last_polled).toLocaleString()}</p>
                                        ${feed.reason ? `<p><strong>Reason:</strong> <span class="reason-text">${this.escapeHtml(feed.reason)}</span></p>` : ''}
                                        ${feed.last_error ? `<p><strong>Error:</strong> <span class="error-text">${this.escapeHtml(feed.last_error)}</span></p>` : ''}
                                        ${feed.user_agent ? `<p><strong>User Agent:</strong> <code>${this.escapeHtml(feed.user_agent)}</code></p>` : ''}
                                        ${stats.non_english_count ? `<p><strong>Non-English:</strong> ${stats.non_english_count}</p>` : ''}
                                        ${stats.avg_content_size ? `<p><strong>Avg Size:</strong> ${this.formatBytes(stats.avg_content_size)}</p>` : ''}
                                    </div>
                                </div>
                            `;
                        }).join('') : '<p>No feeds configured</p>'}
                    </div>
                </div>
            `;
        });
        
        html += '</div>';
        configContainer.innerHTML = html;
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

    formatBytes(bytes, decimals = 2) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const dm = decimals < 0 ? 0 : decimals;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
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