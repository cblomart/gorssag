// Vue.js RSS Aggregator Application
const { createApp, ref, reactive, onMounted, computed } = Vue;

const RSSAggregator = {
    setup() {
        // Reactive state
        const articles = ref([]);
        const topics = ref([]);
        const currentPage = ref(0);
        const pageSize = ref(6); // Will be dynamically calculated based on viewport
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
            const baseArticles = articles.value;
            console.log('🔍 Filtering articles:', {
                totalArticles: baseArticles.length,
                searchQuery: searchQuery.value,
                hasSearch: !!searchQuery.value
            });
            
            if (!searchQuery.value) return baseArticles;
            
            const query = searchQuery.value.toLowerCase();
            const filtered = baseArticles.filter(article => 
                article.title.toLowerCase().includes(query) ||
                article.description?.toLowerCase().includes(query) ||
                article.content?.toLowerCase().includes(query)
            );
            
            console.log('🔍 Search filtered:', {
                originalCount: baseArticles.length,
                filteredCount: filtered.length,
                searchTerm: query
            });
            
            return filtered;
        });

        // Dynamic page size calculation
        const calculateOptimalPageSize = () => {
            // Get viewport dimensions
            const viewportHeight = window.innerHeight;
            const viewportWidth = window.innerWidth;
            
            // Estimate article card dimensions based on CSS
            const estimatedCardHeight = 280; // Approximate height including padding, margins
            const headerHeight = 120; // App header height
            const sidebarOffset = 60; // Additional offset for sidebar and padding
            
            // Calculate available content area height
            const availableHeight = viewportHeight - headerHeight - sidebarOffset;
            
            // Determine grid columns based on viewport width (matching CSS breakpoints)
            let columns = 3; // Default for large screens
            if (viewportWidth <= 480) {
                columns = 1; // Mobile: single column
            } else if (viewportWidth <= 768) {
                columns = 1; // Tablet portrait: single column
            } else if (viewportWidth <= 1200) {
                columns = 2; // Medium screens: 2 columns
            } else {
                columns = 3; // Large screens: 3 columns
            }
            
            // Calculate rows that fit in viewport + some extra for smooth scrolling
            const rowsThatFit = Math.floor(availableHeight / estimatedCardHeight);
            const extraRows = Math.max(1, Math.floor(rowsThatFit * 0.5)); // 50% extra for smooth scroll
            const totalRows = rowsThatFit + extraRows;
            
            // Calculate total articles needed - increased minimum to ensure we see more articles
            const calculatedPageSize = Math.max(12, Math.min(30, totalRows * columns));
            
            console.log('🖥️ Dynamic page size calculation:', {
                viewport: `${viewportWidth}x${viewportHeight}`,
                layout: `${columns} columns`,
                availableHeight: `${availableHeight}px`,
                articlesPerView: `${rowsThatFit} rows × ${columns} cols = ${rowsThatFit * columns}`,
                withBuffering: `+${extraRows * columns} buffer = ${calculatedPageSize} total`,
                calculatedPageSize
            });
            
            return calculatedPageSize;
        };

        // Update page size based on viewport
        const updatePageSize = () => {
            const newPageSize = calculateOptimalPageSize();
            if (newPageSize !== pageSize.value) {
                const oldSize = pageSize.value;
                pageSize.value = newPageSize;
                console.log(`📱 Page size adapted: ${oldSize} → ${newPageSize} articles`);
            }
        };

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

        const loadVersionInfo = async () => {
            try {
                const response = await fetch('/version');
                if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                
                const data = await response.json();
                const versionText = `v${data.version} (${data.git_commit?.substring(0, 7) || 'dev'})`;
                
                // Update the version info in the footer
                const versionElement = document.querySelector('.version-number');
                if (versionElement) {
                    versionElement.textContent = versionText;
                    versionElement.title = `Built: ${data.build_time}\nCommit: ${data.git_commit || 'unknown'}`;
                }
                
                console.log('🏷️ Version Info:', data);
            } catch (error) {
                console.warn('Failed to load version info:', error.message);
                const versionElement = document.querySelector('.version-number');
                if (versionElement) {
                    versionElement.textContent = 'version unknown';
                }
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
                
                console.log('📰 Loaded all articles:', {
                    page,
                    pageSize: pageSize.value,
                    articlesLoadedThisPage: (data.articles || []).length,
                    totalArticlesNow: articles.value.length,
                    hasMore: hasMore.value,
                    totalCountInDB: data.total_count,
                    currentView: currentView.value,
                    sampleTitles: (data.articles || []).slice(0, 3).map(a => a.title?.substring(0, 30) + '...')
                });
                
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
                // Use the working topic feed endpoint instead of broken filter
                const response = await fetch(`/api/v1/feeds/${encodeURIComponent(topic)}?$top=${pageSize.value}&$skip=${skip}&$orderby=published_at desc`);
                if (!response.ok) throw new Error(`HTTP ${response.status}: ${response.statusText}`);
                
                const data = await response.json();
                
                if (page === 0) {
                    articles.value = data.articles || [];
                } else {
                    articles.value = articles.value.concat(data.articles || []);
                }
                
                hasMore.value = data.has_more || false;
                currentPage.value = page;
                
                console.log('Loaded topic articles:', {
                    topic,
                    page,
                    articlesLoaded: (data.articles || []).length,
                    totalArticles: articles.value.length,
                    hasMore: hasMore.value
                });
                
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

        let loadMoreDebounceTimer = null;
        
        const loadMoreArticles = async () => {
            // Debounce rapid scroll events
            if (loadMoreDebounceTimer) {
                clearTimeout(loadMoreDebounceTimer);
            }
            
            loadMoreDebounceTimer = setTimeout(async () => {
                if (isLoadingMore.value || !hasMore.value || loading.value) return;
                
                console.log('Loading more articles, page:', currentPage.value + 1);
                isLoadingMore.value = true;
                
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
            }, 150); // 150ms debounce delay for smooth scrolling
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
            // Calculate initial page size based on viewport
            updatePageSize();
            
            // Set up window resize listener for responsive page size
            const handleResize = () => {
                updatePageSize();
            };
            window.addEventListener('resize', handleResize);
            
            // Load initial data
            loadTopics();
            loadAllArticles();
            loadVersionInfo();
            
            // Setup infinite scroll
            setupInfiniteScroll();
            
            // Setup modal close on outside click
            setupModalHandlers();
            
            // Store resize cleanup function
            resizeCleanup = () => {
                window.removeEventListener('resize', handleResize);
            };
        });

        // Cleanup function
        let resizeCleanup = null;
        const cleanup = () => {
            if (infiniteScrollObserver) {
                infiniteScrollObserver.disconnect();
                infiniteScrollObserver = null;
            }
            if (infiniteScrollTimeout) {
                clearTimeout(infiniteScrollTimeout);
                infiniteScrollTimeout = null;
            }
            if (resizeCleanup) {
                resizeCleanup();
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
                if (!sentinel) {
                    console.log('DEBUG: Infinite scroll sentinel not found');
                    return;
                }

                console.log('DEBUG: Setting up infinite scroll observer', {
                    hasMore: hasMore.value,
                    articlesCount: articles.value.length,
                    sentinelExists: !!sentinel
                });

                infiniteScrollObserver = new IntersectionObserver((entries) => {
                    entries.forEach(entry => {
                        console.log('DEBUG: Intersection observed', {
                            isIntersecting: entry.isIntersecting,
                            intersectionRatio: entry.intersectionRatio,
                            hasMore: hasMore.value,
                            isLoadingMore: isLoadingMore.value,
                            loading: loading.value,
                            articlesCount: articles.value.length
                        });
                        
                        // Only trigger if we have more articles, not loading, and sentinel is visible
                        if (entry.isIntersecting && 
                            hasMore.value && 
                            !isLoadingMore.value && 
                            !loading.value && 
                            articles.value.length > 0) {
                            console.log('Infinite scroll triggered', {
                                hasMore: hasMore.value,
                                isLoadingMore: isLoadingMore.value,
                                loading: loading.value,
                                articlesCount: articles.value.length,
                                intersectionRatio: entry.intersectionRatio,
                                boundingRect: entry.boundingClientRect
                            });
                            loadMoreArticles();
                        }
                    });
                }, {
                    rootMargin: '800px', // Increased from 100px - load content 800px before user reaches the bottom
                    threshold: [0, 0.1, 0.5, 1.0] // Multiple thresholds for better detection
                });

                infiniteScrollObserver.observe(sentinel);
                console.log('DEBUG: Infinite scroll observer set up successfully');
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
            console.log('Showing modal for article ID:', articleId);
            const article = articles.value.find(a => a.id === articleId);
            if (!article) {
                console.error('Article not found with ID:', articleId);
                console.log('Available articles:', articles.value.map(a => a.id));
                return;
            }

            console.log('Found article:', article.title);

            const modal = document.getElementById('articleModal');
            const header = document.getElementById('modalArticleHeader');
            const content = document.getElementById('modalArticleContent');

            if (!modal || !header || !content) {
                console.error('Modal elements not found:', { modal: !!modal, header: !!header, content: !!content });
                return;
            }

            try {
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
                console.log('Modal should now be visible');
            } catch (error) {
                console.error('Error showing modal:', error);
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
            loadVersionInfo,
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
            updatePageSize,
            calculateOptimalPageSize,
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
                        v-if="hasMore && articles.length > 0" 
                        id="infiniteScrollSentinel" 
                        class="infinite-scroll-loading"
                        :class="{ 'loading-active': isLoadingMore }"
                    >
                        <div v-if="isLoadingMore" class="loading-content">
                            <i class="fas fa-spinner fa-spin"></i> 
                            <span>Loading more articles...</span>
                        </div>
                        <div v-else class="loading-trigger">
                            <!-- This invisible element triggers loading when it comes into view -->
                        </div>
                    </div>
                </section>
            </main>

            <!-- Footer with version info -->
            <footer class="app-footer">
                <div class="version-info">
                    <span class="version-text">RSS Aggregator</span>
                    <span class="version-number" ref="versionInfo">Loading version...</span>
                </div>
            </footer>

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