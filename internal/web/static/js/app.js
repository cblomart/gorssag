// RSS Aggregator SPA Application
class RSSAggregator {
    constructor() {
        this.articles = [];
        this.topics = [];
        this.currentPage = 0;
        this.pageSize = 20;
        this.hasMore = true;
        this.loading = false;
        this.currentView = 'all'; // 'all' or 'topic'
        this.selectedTopic = null;
        this.expandedArticles = new Set();
        this.isLoadingMore = false;
    }

    init() {
        // Wait a bit to ensure DOM is fully ready
        setTimeout(() => {
            this.bindEvents();
            this.loadTopics();
            this.loadAllArticles();
            this.setupNavigation();
            this.setupInfiniteScroll();
        }, 100);
    }

    bindEvents() {
        try {
            // Search functionality
            const searchInput = document.getElementById('searchInput');
            if (searchInput) {
                searchInput.addEventListener('input', this.debounce(() => {
                    this.performSearch();
                }, 300));
            }

            // Refresh button
            const refreshBtn = document.getElementById('refreshBtn');
            if (refreshBtn) {
                refreshBtn.addEventListener('click', () => {
                    this.refreshArticles();
                });
            }
        } catch (error) {
            console.error('Error binding events:', error);
        }
    }

    setupInfiniteScroll() {
        this.sentinel = document.getElementById('infiniteScrollSentinel');
        if (!this.sentinel) return;

        this.observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting && this.hasMore && !this.isLoadingMore && !this.loading) {
                    this.loadMoreArticles();
                }
            });
        }, {
            rootMargin: '100px'
        });

        this.updateInfiniteScroll();
    }

    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    async loadTopics() {
        try {
            const response = await fetch('/api/v1/topics');
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            const data = await response.json();
            this.topics = data.topics || [];
            
            // Sort topics alphabetically for consistent order
            this.topics.sort((a, b) => a.localeCompare(b));
            
            this.renderTopics();
        } catch (error) {
            this.showError('Failed to load topics: ' + error.message);
        }
    }

    renderTopics() {
        const topicsContainer = document.getElementById('topicsContainer');
        if (!topicsContainer) return;

        topicsContainer.innerHTML = `
            <div class="topic-item ${this.currentView === 'all' ? 'active' : ''}" onclick="app.showAllArticles()">
                <i class="fas fa-globe"></i>
                <span>All Articles</span>
            </div>
            ${this.topics.map(topic => `
                <div class="topic-item ${this.currentView === 'topic' && this.selectedTopic === topic ? 'active' : ''}" 
                     onclick="app.showTopicArticles('${topic}')">
                    <i class="fas fa-tag"></i>
                    <span>${topic.charAt(0).toUpperCase() + topic.slice(1)}</span>
                </div>
            `).join('')}
        `;
    }

    async loadAllArticles(page = 0) {
        if (this.loading) return;
        
        this.loading = true;
        if (page === 0) {
            this.showLoading();
        }
        
        try {
            const skip = page * this.pageSize;
            const response = await fetch(`/api/v1/articles?$top=${this.pageSize}&$skip=${skip}&$orderby=published_at desc`);
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            const data = await response.json();
            
            if (page === 0) {
                this.articles = data.articles || [];
            } else {
                this.articles = this.articles.concat(data.articles || []);
            }
            
            this.hasMore = data.has_more || false;
            this.currentPage = page;
            
            this.renderArticles();
            this.updateInfiniteScroll();
        } catch (error) {
            this.showError('Failed to load articles: ' + error.message);
        } finally {
            this.loading = false;
            if (page === 0) {
                this.hideLoading();
            }
        }
    }

    async loadTopicArticles(topic, page = 0) {
        if (this.loading) return;
        
        this.loading = true;
        if (page === 0) {
            this.showLoading();
        }
        
        try {
            const skip = page * this.pageSize;
            const response = await fetch(`/api/v1/articles?$filter=topic eq '${encodeURIComponent(topic)}'&$top=${this.pageSize}&$skip=${skip}&$orderby=published_at desc`);
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            const data = await response.json();
            
            if (page === 0) {
                this.articles = data.articles || [];
            } else {
                this.articles = this.articles.concat(data.articles || []);
            }
            
            this.hasMore = data.has_more || false;
            this.currentPage = page;
            
            this.renderArticles();
            this.updateInfiniteScroll();
        } catch (error) {
            this.showError('Failed to load topic articles: ' + error.message);
        } finally {
            this.loading = false;
            if (page === 0) {
                this.hideLoading();
            }
        }
    }

    showAllArticles() {
        this.currentView = 'all';
        this.selectedTopic = null;
        this.currentPage = 0;
        this.articles = [];
        this.hasMore = true;
        this.expandedArticles.clear();
        this.loadAllArticles(0);
        this.renderTopics();
    }

    showTopicArticles(topic) {
        this.currentView = 'topic';
        this.selectedTopic = topic;
        this.currentPage = 0;
        this.articles = [];
        this.hasMore = true;
        this.expandedArticles.clear();
        this.loadTopicArticles(topic, 0);
        this.renderTopics();
    }

    async loadMoreArticles() {
        if (this.isLoadingMore || !this.hasMore || this.loading) return;
        
        this.isLoadingMore = true;
        this.showInfiniteScrollLoading();
        
        try {
            const nextPage = this.currentPage + 1;
            
            if (this.currentView === 'all') {
                await this.loadAllArticles(nextPage);
            } else {
                await this.loadTopicArticles(this.selectedTopic, nextPage);
            }
        } catch (error) {
            this.showError('Failed to load more articles: ' + error.message);
        } finally {
            this.isLoadingMore = false;
            this.hideInfiniteScrollLoading();
        }
    }

    async refreshArticles() {
        if (this.currentView === 'all') {
            this.showAllArticles();
        } else {
            this.showTopicArticles(this.selectedTopic);
        }
    }

    async performSearch() {
        const searchInput = document.getElementById('searchInput');
        if (!searchInput) return;
        
        const query = searchInput.value.trim();
        
        if (query.length === 0) {
            if (this.currentView === 'all') {
                this.showAllArticles();
            } else {
                this.showTopicArticles(this.selectedTopic);
            }
            return;
        }
        
        if (this.loading) return;
        
        this.loading = true;
        this.showLoading();
        
        try {
            const response = await fetch(`/api/v1/articles?$search=${encodeURIComponent(query)}&$top=50&$orderby=published_at desc`);
            if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            
            const data = await response.json();
            this.articles = data.articles || [];
            this.hasMore = false; // Disable infinite scroll for search results
            this.currentPage = 0;
            this.expandedArticles.clear();
            
            this.renderArticles();
            this.updateInfiniteScroll();
        } catch (error) {
            this.showError('Search failed: ' + error.message);
        } finally {
            this.loading = false;
            this.hideLoading();
        }
    }

    // Render articles
    renderArticles() {
        const articlesContainer = document.getElementById('articlesContainer');
        if (!articlesContainer) {
            console.error('Articles container not found');
            return;
        }

        if (this.articles.length === 0) {
            articlesContainer.innerHTML = `
                <div style="grid-column: 1 / -1; text-align: center; padding: 3rem; color: #666;">
                    <i class="fas fa-newspaper" style="font-size: 3rem; margin-bottom: 1rem; color: #ccc;"></i>
                    <p>No articles found</p>
                </div>
            `;
            return;
        }

        const articlesHTML = this.articles.map((article, index) => {
            // Validate article data
            if (!article.id) {
                console.error(`Article at index ${index} missing ID:`, article);
                return '';
            }
            
            if (!article.title) {
                console.error(`Article ${article.id} missing title`);
                return '';
            }

            const preview = this.getArticlePreview(article.content);
            const hasLink = article.link && article.link.trim() !== '';
            
            return `
                <div class="article-card" data-article-id="${article.id}">
                    <div class="article-header">
                        <div class="article-title">${this.escapeHtml(article.title)}</div>
                        <div class="article-meta">
                            <span class="article-source">${this.escapeHtml(article.source || 'Unknown')}</span>
                            <span class="article-date">${new Date(article.published_at).toLocaleDateString()}</span>
                            <span class="article-topic">${this.escapeHtml(article.topic || 'Unknown')}</span>
                            ${article.language && article.language !== 'en' ? `<span class="language-badge" title="Language: ${this.getLanguageName(article.language)}">${article.language.toUpperCase()}</span>` : ''}
                        </div>
                    </div>
                    <div class="article-content collapsed">
                        ${preview}
                    </div>
                    <div class="article-actions">
                        <button class="readmore-button" data-article-id="${article.id}" type="button">
                            Read More
                        </button>
                        ${hasLink ? `<a href="${this.escapeHtml(article.link)}" target="_blank" class="read-more-link">Read Full Article</a>` : '<span class="no-link">No link available</span>'}
                    </div>
                </div>
            `;
        }).join('');

        articlesContainer.innerHTML = articlesHTML;
        this.bindArticleEvents();
    }

    // Bind events specifically for articles and modal
    bindArticleEvents() {
        const articlesContainer = document.getElementById('articlesContainer');
        if (!articlesContainer) {
            console.error('Articles container not found for event binding');
            return;
        }

        // Remove any existing event listeners
        const readMoreButtons = articlesContainer.querySelectorAll('.readmore-button');
        
        readMoreButtons.forEach((button, index) => {
            const newButton = button.cloneNode(true);
            button.parentNode.replaceChild(newButton, button);
        });

        // Add new event listeners
        const newReadMoreButtons = articlesContainer.querySelectorAll('.readmore-button');
        
        newReadMoreButtons.forEach((button, index) => {
            const articleId = button.getAttribute('data-article-id');
            
            button.addEventListener('click', (event) => {
                event.preventDefault();
                event.stopPropagation();
                
                if (articleId) {
                    this.showArticleModal(articleId);
                } else {
                    console.error('No article ID found for clicked button');
                }
            });
        });

        // Modal close button
        const modal = document.getElementById('articleModal');
        const closeBtn = document.getElementById('modalCloseBtn');
        if (closeBtn) {
            closeBtn.onclick = () => this.hideArticleModal();
        }
        // Click outside modal content
        if (modal) {
            modal.onclick = (e) => {
                if (e.target === modal) this.hideArticleModal();
            };
        }
        // Escape key
        document.onkeydown = (e) => {
            if (e.key === 'Escape') this.hideArticleModal();
        };
    }

    // Show article in modal
    showArticleModal(articleId) {
        const article = this.articles.find(a => a.id === articleId);
        if (!article) return;
        const modal = document.getElementById('articleModal');
        const modalHeader = document.getElementById('modalArticleHeader');
        const modalContent = document.getElementById('modalArticleContent');
        if (!modal || !modalHeader || !modalContent) return;
        
        // Set modal header with article metadata
        modalHeader.innerHTML = `
            <div class="modal-article-title">${this.escapeHtml(article.title)}</div>
            <div class="modal-article-meta">
                <span class="modal-article-source">${this.escapeHtml(article.source)}</span>
                <span class="modal-article-date">${new Date(article.published_at).toLocaleDateString()}</span>
                <span class="modal-article-topic">${this.escapeHtml(article.topic)}</span>
            </div>
        `;
        
        // Set modal content
        modalContent.innerHTML = this.renderMarkdown(article.content);
        modal.classList.add('show');
        modal.style.display = 'flex';
        document.body.style.overflow = 'hidden';
    }

    // Hide modal
    hideArticleModal() {
        const modal = document.getElementById('articleModal');
        if (!modal) return;
        modal.classList.remove('show');
        modal.style.display = 'none';
        document.body.style.overflow = '';
    }

    // Get article preview (first few lines)
    getArticlePreview(content) {
        if (!content) return 'No content available';
        
        // Clean up the content and get first few lines
        const lines = content.split('\n').filter(line => line.trim() !== '');
        const previewLines = lines.slice(0, 3); // Show first 3 non-empty lines
        
        return this.renderMarkdown(previewLines.join('\n'));
    }

    // Render markdown content
    renderMarkdown(content) {
        if (!content) return 'No content available';
        
        // Clean up empty links first
        let cleanedContent = content
            .replace(/\[\]\([^)]*\)/g, '') // Remove empty links like [](http://...)
            .replace(/\[\[([^\]]*)\]\]\(([^)]*)\)/g, '[$1]($2)') // Fix double brackets
            .replace(/\[[^\]]*\]\([^)]*\)/g, (match) => {
                // Extract link text and URL
                const linkMatch = match.match(/\[([^\]]*)\]\(([^)]*)\)/);
                if (linkMatch && linkMatch[1].trim() === '') {
                    return ''; // Remove links with empty text
                }
                return match; // Keep links with text
            });
        
        // Simple markdown rendering
        let html = cleanedContent
            // Headers
            .replace(/^### (.*$)/gim, '<h3>$1</h3>')
            .replace(/^## (.*$)/gim, '<h2>$1</h2>')
            .replace(/^# (.*$)/gim, '<h1>$1</h1>')
            // Bold
            .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
            // Italic
            .replace(/\*(.*?)\*/g, '<em>$1</em>')
            // Links
            .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank">$1</a>')
            // Lists
            .replace(/^- (.*$)/gim, '<li>$1</li>')
            .replace(/^(\d+)\. (.*$)/gim, '<li>$2</li>')
            // Code blocks
            .replace(/```([\s\S]*?)```/g, '<pre><code>$1</code></pre>')
            .replace(/`([^`]+)`/g, '<code>$1</code>')
            // Blockquotes
            .replace(/^> (.*$)/gim, '<blockquote>$1</blockquote>')
            // Line breaks - handle them properly
            .replace(/\n\n/g, '</p><p>')
            .replace(/\n/g, '<br>');
        
        // Wrap in paragraph tags
        html = '<p>' + html + '</p>';
        
        // Clean up empty paragraphs and fix list structure
        html = html
            .replace(/<p><\/p>/g, '')
            .replace(/<p>(<h[1-6]>.*?<\/h[1-6]>)<\/p>/g, '$1')
            .replace(/<p>(<blockquote>.*?<\/blockquote>)<\/p>/g, '$1')
            .replace(/<p>(<pre>.*?<\/pre>)<\/p>/g, '$1')
            .replace(/<p>(<li>.*?<\/li>)<\/p>/g, '<ul>$1</ul>');
        
        return html;
    }

    updateInfiniteScroll() {
        if (!this.sentinel) return;
        
        if (this.hasMore) {
            this.sentinel.style.display = 'block';
            this.observer.observe(this.sentinel);
        } else {
            this.sentinel.style.display = 'none';
            this.observer.unobserve(this.sentinel);
        }
    }

    showInfiniteScrollLoading() {
        if (this.sentinel) {
            this.sentinel.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Loading more articles...';
            this.sentinel.style.display = 'block';
        }
    }

    hideInfiniteScrollLoading() {
        if (this.sentinel) {
            this.sentinel.innerHTML = '';
            this.sentinel.style.display = 'none';
        }
    }

    setupNavigation() {
        // Add any navigation setup here
    }

    showLoading() {
        const loadingEl = document.getElementById('loading');
        if (loadingEl) loadingEl.style.display = 'block';
    }

    hideLoading() {
        const loadingEl = document.getElementById('loading');
        if (loadingEl) loadingEl.style.display = 'none';
    }

    showError(message) {
        const errorEl = document.getElementById('error');
        if (errorEl) {
            errorEl.textContent = message;
            errorEl.style.display = 'block';
            setTimeout(() => {
                errorEl.style.display = 'none';
            }, 5000);
        }
        console.error(message);
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString();
    }

    getLanguageName(languageCode) {
        const languageNames = {
            'en': 'English',
            'de': 'German',
            'fr': 'French',
            'es': 'Spanish',
            'zh': 'Chinese',
            'ru': 'Russian',
            'it': 'Italian',
            'pt': 'Portuguese',
            'nl': 'Dutch',
            'sv': 'Swedish',
            'da': 'Danish',
            'fi': 'Finnish',
            'pl': 'Polish',
            'cs': 'Czech',
            'hu': 'Hungarian',
            'ro': 'Romanian'
        };
        return languageNames[languageCode] || languageCode.toUpperCase();
    }
}

// Initialize the app
let app;

function initializeApp() {
    try {
        app = new RSSAggregator();
        app.init();
        // Make app globally accessible
        window.app = app;
    } catch (error) {
        console.error('Failed to initialize RSS Aggregator:', error);
    }
}

// Wait for DOM to be ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeApp);
} else {
    // DOM is already ready
    initializeApp();
} 