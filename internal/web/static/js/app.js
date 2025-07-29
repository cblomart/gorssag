// RSS Aggregator SPA Application
class RSSAggregatorApp {
    constructor() {
        this.currentView = 'topics';
        this.topics = [];
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadTopics();
        this.setupNavigation();
    }

    bindEvents() {
        // Navigation
        document.getElementById('topicsBtn').addEventListener('click', () => this.showView('topics'));
        document.getElementById('searchBtn').addEventListener('click', () => this.showView('search'));
        document.getElementById('docsBtn').addEventListener('click', () => this.showView('docs'));

        // Topics
        document.getElementById('refreshTopics').addEventListener('click', () => this.loadTopics());

        // Search
        document.getElementById('searchSubmit').addEventListener('click', () => this.performSearch());
        document.getElementById('searchInput').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.performSearch();
        });

        // Modal
        document.querySelector('.modal-close').addEventListener('click', () => this.hideErrorModal());
        document.querySelector('.modal').addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) this.hideErrorModal();
        });
    }

    setupNavigation() {
        // Load topics into search filter
        this.loadTopicsForFilter();
    }

    showView(viewName) {
        // Update navigation
        document.querySelectorAll('.nav-btn').forEach(btn => btn.classList.remove('active'));
        document.getElementById(viewName + 'Btn').classList.add('active');

        // Update views
        document.querySelectorAll('.view').forEach(view => view.classList.remove('active'));
        document.getElementById(viewName + 'View').classList.add('active');

        this.currentView = viewName;

        // Load data for view
        if (viewName === 'topics') {
            this.loadTopics();
        } else if (viewName === 'search') {
            this.loadTopicsForFilter();
        }
    }

    async loadTopics() {
        this.showLoading();
        try {
            const response = await fetch('/api/v1/topics');
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            const topics = await response.json();
            this.topics = topics;
            this.renderTopics(topics);
        } catch (error) {
            this.showError('Failed to load topics: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    async loadTopicsForFilter() {
        try {
            const response = await fetch('/api/v1/topics');
            if (!response.ok) return;
            
            const topics = await response.json();
            this.populateTopicFilter(topics);
        } catch (error) {
            console.error('Failed to load topics for filter:', error);
        }
    }

    renderTopics(topics) {
        const container = document.getElementById('topicsList');
        
        if (topics.length === 0) {
            container.innerHTML = `
                <div class="text-center" style="grid-column: 1 / -1; padding: 2rem;">
                    <p class="text-muted">No topics available</p>
                </div>
            `;
            return;
        }

        container.innerHTML = topics.map(topic => `
            <div class="topic-card" onclick="app.loadTopicArticles('${topic}')">
                <h3><i class="fas fa-folder"></i> ${this.capitalizeFirst(topic)}</h3>
                <div class="topic-meta">
                    <span><i class="fas fa-rss"></i> RSS Feed</span>
                </div>
                <div class="topic-actions">
                    <button class="btn btn-outline" onclick="event.stopPropagation(); app.loadTopicArticles('${topic}')">
                        <i class="fas fa-eye"></i> View Articles
                    </button>
                    <button class="btn btn-secondary" onclick="event.stopPropagation(); app.refreshTopic('${topic}')">
                        <i class="fas fa-sync-alt"></i> Refresh
                    </button>
                </div>
            </div>
        `).join('');
    }

    populateTopicFilter(topics) {
        const select = document.getElementById('topicFilter');
        select.innerHTML = '<option value="">All Topics</option>';
        
        topics.forEach(topic => {
            const option = document.createElement('option');
            option.value = topic;
            option.textContent = this.capitalizeFirst(topic);
            select.appendChild(option);
        });
    }

    async loadTopicArticles(topic) {
        this.showLoading();
        try {
            const response = await fetch(`/api/v1/feeds/${topic}`);
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            const feed = await response.json();
            this.showSearchViewWithTopic(topic, feed.articles);
        } catch (error) {
            this.showError(`Failed to load articles for ${topic}: ${error.message}`);
        } finally {
            this.hideLoading();
        }
    }

    async refreshTopic(topic) {
        this.showLoading();
        try {
            const response = await fetch(`/api/v1/feeds/${topic}/refresh`, { method: 'POST' });
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            // Reload the topic articles
            await this.loadTopicArticles(topic);
        } catch (error) {
            this.showError(`Failed to refresh ${topic}: ${error.message}`);
        } finally {
            this.hideLoading();
        }
    }

    showSearchViewWithTopic(topic, articles) {
        this.showView('search');
        document.getElementById('topicFilter').value = topic;
        this.renderArticles(articles);
    }

    async performSearch() {
        const searchTerm = document.getElementById('searchInput').value.trim();
        const topic = document.getElementById('topicFilter').value;
        const sortBy = document.getElementById('sortBy').value;
        const limit = document.getElementById('limitResults').value;

        if (!searchTerm && !topic) {
            this.showError('Please enter a search term or select a topic');
            return;
        }

        this.showLoading();
        try {
            const params = new URLSearchParams();
            if (searchTerm) params.append('$search', searchTerm);
            if (topic) params.append('topic', topic);
            if (sortBy) params.append('$orderby', sortBy);
            if (limit) params.append('$top', limit);

            const url = topic ? `/api/v1/feeds/${topic}` : '/api/v1/search';
            const response = await fetch(`${url}?${params.toString()}`);
            
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            const data = await response.json();
            this.renderArticles(data.articles || data);
        } catch (error) {
            this.showError('Search failed: ' + error.message);
        } finally {
            this.hideLoading();
        }
    }

    renderArticles(articles) {
        const container = document.getElementById('searchResults');
        
        if (!articles || articles.length === 0) {
            container.innerHTML = `
                <div class="text-center" style="padding: 2rem;">
                    <p class="text-muted">No articles found</p>
                </div>
            `;
            return;
        }

        container.innerHTML = articles.map(article => `
            <div class="article-card">
                <h3>
                    <a href="${article.link}" target="_blank" rel="noopener noreferrer">
                        ${this.escapeHtml(article.title)}
                    </a>
                </h3>
                <div class="article-meta">
                    ${article.author ? `<span><i class="fas fa-user"></i> ${this.escapeHtml(article.author)}</span>` : ''}
                    ${article.source ? `<span><i class="fas fa-globe"></i> ${this.escapeHtml(article.source)}</span>` : ''}
                    ${article.published_at ? `<span><i class="fas fa-calendar"></i> ${this.formatDate(article.published_at)}</span>` : ''}
                </div>
                ${article.description ? `<div class="article-description">${this.escapeHtml(article.description)}</div>` : ''}
                ${article.categories && article.categories.length > 0 ? `
                    <div class="article-categories">
                        ${article.categories.map(cat => `<span class="category-tag">${this.escapeHtml(cat)}</span>`).join('')}
                    </div>
                ` : ''}
            </div>
        `).join('');
    }

    // Utility methods
    showLoading() {
        document.getElementById('loadingOverlay').classList.remove('hidden');
    }

    hideLoading() {
        document.getElementById('loadingOverlay').classList.add('hidden');
    }

    showError(message) {
        document.getElementById('errorMessage').textContent = message;
        document.getElementById('errorModal').classList.remove('hidden');
    }

    hideErrorModal() {
        document.getElementById('errorModal').classList.add('hidden');
    }

    capitalizeFirst(str) {
        return str.charAt(0).toUpperCase() + str.slice(1);
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
    }
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new RSSAggregatorApp();
}); 