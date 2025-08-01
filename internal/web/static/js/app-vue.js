// Vue.js RSS Aggregator Application
const { createApp, ref, reactive, onMounted, computed } = Vue;

const RSSAggregator = {
    setup() {
        // Reactive state
        const articles = ref([]);
        const topics = ref([]);
        const currentPage = ref(0);
        const pageSize = ref(20);
        const hasMore = ref(true);
        const loading = ref(false);
        const currentView = ref('all');
        const selectedTopic = ref(null);
        const expandedArticles = ref(new Set());
        const isLoadingMore = ref(false);
        const searchQuery = ref('');
        const errorMessage = ref('');

        // Computed properties
        const filteredArticles = computed(() => {
            if (!searchQuery.value) return articles.value;
            
            const query = searchQuery.value.toLowerCase();
            return articles.value.filter(article => 
                article.title.toLowerCase().includes(query) ||
                article.description?.toLowerCase().includes(query) ||
                article.content?.toLowerCase().includes(query)
            );
        });

        // Methods
        const loadTopics = async () => {
            try {
                const response = await fetch('/api/v1/topics');
                if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                
                const data = await response.json();
                topics.value = (data.topics || []).sort((a, b) => a.localeCompare(b));
            } catch (error) {
                showError('Failed to load topics: ' + error.message);
            }
        };

        const loadAllArticles = async (page = 0) => {
            if (loading.value) return;
            
            loading.value = true;
            
            try {
                const skip = page * pageSize.value;
                const response = await fetch(`/api/v1/articles?$top=${pageSize.value}&$skip=${skip}&$orderby=published_at desc`);
                if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                
                const data = await response.json();
                
                if (page === 0) {
                    articles.value = data.articles || [];
                } else {
                    articles.value = articles.value.concat(data.articles || []);
                }
                
                hasMore.value = data.has_more || false;
                currentPage.value = page;
                
                // Reset infinite scroll after loading articles
                if (page === 0) {
                    watchArticles();
                }
            } catch (error) {
                showError('Failed to load articles: ' + error.message);
            } finally {
                loading.value = false;
            }
        };

        const loadTopicArticles = async (topic, page = 0) => {
            if (loading.value) return;
            
            loading.value = true;
            
            try {
                const skip = page * pageSize.value;
                const response = await fetch(`/api/v1/articles?$filter=topic eq '${encodeURIComponent(topic)}'&$top=${pageSize.value}&$skip=${skip}&$orderby=published_at desc`);
                if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                
                const data = await response.json();
                
                if (page === 0) {
                    articles.value = data.articles || [];
                } else {
                    articles.value = articles.value.concat(data.articles || []);
                }
                
                hasMore.value = data.has_more || false;
                currentPage.value = page;
                
                // Reset infinite scroll after loading articles
                if (page === 0) {
                    watchArticles();
                }
            } catch (error) {
                showError('Failed to load topic articles: ' + error.message);
            } finally {
                loading.value = false;
            }
        };

        const showAllArticles = () => {
            currentView.value = 'all';
            selectedTopic.value = null;
            loadAllArticles(0);
            watchCurrentView();
        };

        const showTopicArticles = (topic) => {
            currentView.value = 'topic';
            selectedTopic.value = topic;
            loadTopicArticles(topic, 0);
            watchCurrentView();
        };

        const loadMoreArticles = async () => {
            if (isLoadingMore.value || !hasMore.value) return;
            
            const nextPage = currentPage.value + 1;
            
            try {
                if (currentView.value === 'all') {
                    await loadAllArticles(nextPage);
                } else {
                    await loadTopicArticles(selectedTopic.value, nextPage);
                }
            } finally {
                isLoadingMore.value = false;
            }
        };

        const refreshArticles = async () => {
            if (currentView.value === 'all') {
                await loadAllArticles(0);
            } else {
                await loadTopicArticles(selectedTopic.value, 0);
            }
        };

        const performSearch = () => {
            // Search is handled by computed property
        };

        const getArticlePreview = (content) => {
            if (!content) return '';
            const maxLength = 200;
            const text = content.replace(/<[^>]*>/g, '').trim();
            return text.length > maxLength ? text.substring(0, maxLength) + '...' : text;
        };

        const escapeHtml = (text) => {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        };

        const formatDate = (dateString) => {
            return new Date(dateString).toLocaleDateString();
        };

        const getLanguageName = (languageCode) => {
            const languages = {
                'en': 'English',
                'es': 'Spanish',
                'fr': 'French',
                'de': 'German',
                'it': 'Italian',
                'pt': 'Portuguese',
                'ru': 'Russian',
                'zh': 'Chinese',
                'ja': 'Japanese',
                'ko': 'Korean'
            };
            return languages[languageCode] || languageCode.toUpperCase();
        };

        const showError = (message) => {
            errorMessage.value = message;
            setTimeout(() => {
                errorMessage.value = '';
            }, 5000);
        };

        const capitalizeFirst = (str) => {
            return str.charAt(0).toUpperCase() + str.slice(1);
        };

        // Lifecycle
        onMounted(() => {
            loadTopics();
            loadAllArticles();
            
            // Setup infinite scroll
            setupInfiniteScroll();
            
            // Setup modal close on outside click
            setupModalHandlers();
        });

        // Cleanup function
        const cleanup = () => {
            if (infiniteScrollObserver) {
                infiniteScrollObserver.disconnect();
                infiniteScrollObserver = null;
            }
            if (infiniteScrollTimeout) {
                clearTimeout(infiniteScrollTimeout);
                infiniteScrollTimeout = null;
            }
        };

        let infiniteScrollObserver = null;
        let infiniteScrollTimeout = null;

        const setupInfiniteScroll = () => {
            // Clean up existing observer
            if (infiniteScrollObserver) {
                infiniteScrollObserver.disconnect();
                infiniteScrollObserver = null;
            }

            // Clear any existing timeout
            if (infiniteScrollTimeout) {
                clearTimeout(infiniteScrollTimeout);
                infiniteScrollTimeout = null;
            }

            // Wait for next tick to ensure DOM is updated
            infiniteScrollTimeout = setTimeout(() => {
                const sentinel = document.getElementById('infiniteScrollSentinel');
                if (!sentinel) return;

                infiniteScrollObserver = new IntersectionObserver((entries) => {
                    entries.forEach(entry => {
                        // Only trigger if we have more articles, not loading, and sentinel is visible
                        if (entry.isIntersecting && 
                            hasMore.value && 
                            !isLoadingMore.value && 
                            !loading.value && 
                            articles.value.length > 0) {
                            // Set loading state immediately to prevent multiple triggers
                            isLoadingMore.value = true;
                            loadMoreArticles();
                        }
                    });
                }, {
                    rootMargin: '100px'
                });

                infiniteScrollObserver.observe(sentinel);
            }, 100);
        };

        const setupModalHandlers = () => {
            // Close modal when clicking outside
            document.addEventListener('click', (e) => {
                const modal = document.getElementById('articleModal');
                if (modal && e.target === modal) {
                    closeModal();
                }
            });

            // Close modal with Escape key
            document.addEventListener('keydown', (e) => {
                if (e.key === 'Escape') {
                    closeModal();
                }
            });
        };

        const showArticleModal = (articleId) => {
            const article = articles.value.find(a => a.id === articleId);
            if (!article) return;

            const modal = document.getElementById('articleModal');
            const header = document.getElementById('modalArticleHeader');
            const content = document.getElementById('modalArticleContent');

            if (modal && header && content) {
                header.innerHTML = `
                    <h2>${escapeHtml(article.title)}</h2>
                    <div class="modal-meta">
                        <span class="modal-source">${escapeHtml(article.source || 'Unknown')}</span>
                        <span class="modal-date">${formatDate(article.published_at)}</span>
                        <span class="modal-topic">${escapeHtml(article.topic || 'Unknown')}</span>
                    </div>
                `;
                
                content.innerHTML = `
                    <div class="modal-content-text">${article.content || 'No content available'}</div>
                    ${article.link ? `<a href="${escapeHtml(article.link)}" target="_blank" class="modal-link">Read Full Article</a>` : ''}
                `;
                
                modal.style.display = 'block';
            }
        };

        const closeModal = () => {
            const modal = document.getElementById('articleModal');
            if (modal) {
                modal.style.display = 'none';
            }
        };

        // Watch for changes that require infinite scroll re-setup
        const watchForInfiniteScrollReset = () => {
            setupInfiniteScroll();
        };

        // Watch articles changes to reset infinite scroll
        const watchArticles = () => {
            if (articles.value.length > 0) {
                setTimeout(() => {
                    setupInfiniteScroll();
                }, 100);
            }
        };

        // Watch current view changes to reset infinite scroll
        const watchCurrentView = () => {
            setTimeout(() => {
                setupInfiniteScroll();
            }, 100);
        };

        return {
            // State
            articles,
            topics,
            currentPage,
            pageSize,
            hasMore,
            loading,
            currentView,
            selectedTopic,
            expandedArticles,
            isLoadingMore,
            searchQuery,
            errorMessage,
            
            // Computed
            filteredArticles,
            
            // Methods
            loadTopics,
            loadAllArticles,
            loadTopicArticles,
            showAllArticles,
            showTopicArticles,
            loadMoreArticles,
            refreshArticles,
            performSearch,
            showArticleModal,
            getArticlePreview,
            escapeHtml,
            formatDate,
            getLanguageName,
            showError,
            capitalizeFirst,
            setupInfiniteScroll,
            closeModal,
            watchForInfiniteScrollReset,
            setupModalHandlers,
            watchArticles,
            watchCurrentView,
            cleanup
        };
    },

    template: `
        <div class="app-container">
            <!-- Header -->
            <header class="app-header">
                <div class="header-content">
                    <h1><i class="fas fa-newspaper"></i> RSS Aggregator</h1>
                    <nav class="header-nav">
                        <a href="#" class="nav-link" :class="{ active: currentView === 'all' }" @click="showAllArticles">
                            <i class="fas fa-home"></i> Home
                        </a>
                        <a href="/config" class="nav-link">
                            <i class="fas fa-cog"></i> Configuration
                        </a>
                    </nav>
                    <div class="header-actions">
                        <div class="search-container">
                            <input 
                                type="text" 
                                v-model="searchQuery"
                                placeholder="Search articles..." 
                                class="search-input"
                                @input="performSearch"
                            >
                            <i class="fas fa-search search-icon"></i>
                        </div>
                        <button @click="refreshArticles" class="refresh-button" :disabled="loading">
                            <i class="fas fa-sync-alt" :class="{ 'fa-spin': loading }"></i>
                        </button>
                    </div>
                </div>
            </header>

            <!-- Main Content -->
            <main class="app-main">
                <!-- Sidebar -->
                <aside class="sidebar">
                    <div class="sidebar-header">
                        <h3>Topics</h3>
                    </div>
                    <div class="topics-list">
                        <div 
                            class="topic-item" 
                            :class="{ active: currentView === 'all' }" 
                            @click="showAllArticles"
                        >
                            <i class="fas fa-globe"></i>
                            <span>All Articles</span>
                        </div>
                        <div 
                            v-for="topic in topics" 
                            :key="topic"
                            class="topic-item" 
                            :class="{ active: currentView === 'topic' && selectedTopic === topic }" 
                            @click="showTopicArticles(topic)"
                        >
                            <i class="fas fa-tag"></i>
                            <span>{{ capitalizeFirst(topic) }}</span>
                        </div>
                    </div>
                </aside>

                <!-- Content Area -->
                <section class="content-area">
                    <!-- Loading Indicator -->
                    <div v-if="loading" class="loading-overlay">
                        <div class="loading-spinner">
                            <i class="fas fa-spinner fa-spin"></i>
                            <span>Loading...</span>
                        </div>
                    </div>

                    <!-- Error Display -->
                    <div v-if="errorMessage" class="error-message">
                        <i class="fas fa-exclamation-triangle"></i>
                        {{ errorMessage }}
                    </div>

                    <!-- Articles Container -->
                    <div v-if="!loading && filteredArticles.length === 0" class="articles-container">
                        <div style="grid-column: 1 / -1; text-align: center; padding: 3rem; color: #666;">
                            <i class="fas fa-newspaper" style="font-size: 3rem; margin-bottom: 1rem; color: #ccc;"></i>
                            <p>No articles found</p>
                        </div>
                    </div>

                    <div v-else class="articles-container">
                        <div 
                            v-for="article in filteredArticles" 
                            :key="article.id"
                            class="article-card" 
                            :data-article-id="article.id"
                        >
                            <div class="article-header">
                                <div class="article-title">{{ escapeHtml(article.title) }}</div>
                                <div class="article-meta">
                                    <span class="article-source">{{ escapeHtml(article.source || 'Unknown') }}</span>
                                    <span class="article-date">{{ formatDate(article.published_at) }}</span>
                                    <span class="article-topic">{{ escapeHtml(article.topic || 'Unknown') }}</span>
                                    <span 
                                        v-if="article.language && article.language !== 'en'" 
                                        class="language-badge" 
                                        :title="'Language: ' + getLanguageName(article.language)"
                                    >
                                        {{ article.language.toUpperCase() }}
                                    </span>
                                </div>
                            </div>
                            <div class="article-content collapsed">
                                {{ getArticlePreview(article.content) }}
                            </div>
                            <div class="article-actions">
                                <button 
                                    class="readmore-button" 
                                    :data-article-id="article.id" 
                                    type="button"
                                    @click="showArticleModal(article.id)"
                                >
                                    Read More
                                </button>
                                <a 
                                    v-if="article.link && article.link.trim()" 
                                    :href="escapeHtml(article.link)" 
                                    target="_blank" 
                                    class="read-more-link"
                                >
                                    Read Full Article
                                </a>
                                <span v-else class="no-link">No link available</span>
                            </div>
                        </div>
                    </div>
                    
                    <!-- Infinite Scroll Sentinel -->
                    <div 
                        v-if="hasMore && !loading && !isLoadingMore && articles.length > 0" 
                        id="infiniteScrollSentinel" 
                        class="infinite-scroll-loading"
                    >
                        <i class="fas fa-spinner fa-spin"></i> Loading more articles...
                    </div>
                </section>
            </main>

            <!-- Article Modal -->
            <div id="articleModal" class="modal" style="display: none;">
                <div class="modal-content">
                    <span class="modal-close" @click="closeModal">&times;</span>
                    <div id="modalArticleHeader"></div>
                    <div id="modalArticleContent"></div>
                </div>
            </div>
        </div>
    `
};

// Initialize Vue app
const app = createApp(RSSAggregator);
app.mount('#app'); 