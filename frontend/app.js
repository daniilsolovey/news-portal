const RPC_URL = 'http://localhost:3000/rpc';
let rpcIdCounter = 1;

// Helper function to display results
function displayResult(elementId, data, isError = false) {
    const resultDiv = document.getElementById(elementId);
    resultDiv.className = isError ? 'result error' : 'result';
    
    if (isError) {
        resultDiv.innerHTML = `<pre>${JSON.stringify(data, null, 2)}</pre>`;
        return;
    }

    if (data === null || data === undefined) {
        resultDiv.innerHTML = '<div class="empty">Нет данных</div>';
        return;
    }

    // Format JSON with syntax highlighting
    resultDiv.innerHTML = `<pre>${JSON.stringify(data, null, 2)}</pre>`;
}

// Helper function to show loading
function showLoading(elementId) {
    const resultDiv = document.getElementById(elementId);
    resultDiv.className = 'result';
    resultDiv.innerHTML = '<div class="loading">Загрузка...</div>';
}

// JSON-RPC call helper
async function jsonRpcCall(method, params, elementId, customFormatter = null) {
    showLoading(elementId);
    try {
        const request = {
            jsonrpc: "2.0",
            method: method,
            params: params,
            id: rpcIdCounter++
        };

        const response = await fetch(RPC_URL, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(request)
        });

        const data = await response.json();
        
        if (data.error) {
            displayResult(elementId, { 
                error: data.error.message || 'Ошибка RPC', 
                code: data.error.code,
                data: data.error.data 
            }, true);
            return;
        }

        if (customFormatter) {
            customFormatter(elementId, data.result);
        } else {
            displayResult(elementId, data.result);
        }
    } catch (error) {
        displayResult(elementId, { error: error.message || 'Ошибка сети' }, true);
    }
}

// Get all news
async function getAllNews() {
    const tagId = document.getElementById('allNewsTagId').value;
    const categoryId = document.getElementById('allNewsCategoryId').value;
    const page = document.getElementById('allNewsPage').value || 1;
    const pageSize = document.getElementById('allNewsPageSize').value || 10;

    const filter = {};
    if (tagId) filter.tagId = parseInt(tagId);
    if (categoryId) filter.categoryId = parseInt(categoryId);
    if (page) filter.page = parseInt(page);
    if (pageSize) filter.pageSize = parseInt(pageSize);

    jsonRpcCall('news.List', { filter }, 'allNewsResult', (elementId, data) => {
        const resultDiv = document.getElementById(elementId);
        resultDiv.className = 'result';
        
        if (!data || !Array.isArray(data) || data.length === 0) {
            resultDiv.innerHTML = '<div class="empty">Новости не найдены</div>';
            return;
        }

        let html = `<div style="margin-bottom: 15px;"><strong>Найдено новостей: ${data.length}</strong></div>`;
        data.forEach(news => {
            const publishedDate = new Date(news.publishedAt).toLocaleString('ru-RU');
            const tagsHtml = (news.tags || []).map(tag => `<span class="tag">${tag.title}</span>`).join('');
            
            html += `
                <div class="news-item">
                    <h3>${news.title}</h3>
                    <div class="meta">
                        <strong>Автор:</strong> ${news.author} | 
                        <strong>Категория:</strong> ${news.category ? news.category.title : 'N/A'} | 
                        <strong>Опубликовано:</strong> ${publishedDate}
                    </div>
                    <div class="tags">${tagsHtml}</div>
                </div>
            `;
        });
        
        html += `<pre style="margin-top: 20px;">${JSON.stringify(data, null, 2)}</pre>`;
        resultDiv.innerHTML = html;
    });
}

// Get news count
async function getNewsCount() {
    const tagId = document.getElementById('countTagId').value;
    const categoryId = document.getElementById('countCategoryId').value;

    const filter = {};
    if (tagId) filter.tagId = parseInt(tagId);
    if (categoryId) filter.categoryId = parseInt(categoryId);

    jsonRpcCall('news.Count', { filter }, 'countResult', (elementId, count) => {
        const resultDiv = document.getElementById(elementId);
        resultDiv.className = 'result';
        resultDiv.innerHTML = `
            <div style="text-align: center; padding: 20px;">
                <h2 style="color: #667eea; font-size: 2.5em; margin-bottom: 10px;">${count}</h2>
                <p style="color: #666;">новостей найдено</p>
            </div>
            <pre>${JSON.stringify(count, null, 2)}</pre>
        `;
    });
}

// Get news by ID
async function getNewsById() {
    const newsId = document.getElementById('newsId').value;
    
    if (!newsId) {
        displayResult('newsByIdResult', { error: 'Введите ID новости' }, true);
        return;
    }

    jsonRpcCall('news.ByID', { req: { id: parseInt(newsId) } }, 'newsByIdResult', (elementId, data) => {
        const resultDiv = document.getElementById(elementId);
        resultDiv.className = 'result';
        
        if (!data) {
            resultDiv.innerHTML = '<div class="empty">Новость не найдена</div>';
            return;
        }
        
        const publishedDate = new Date(data.publishedAt).toLocaleString('ru-RU');
        const tagsHtml = (data.tags || []).map(tag => `<span class="tag">${tag.title}</span>`).join('');
        
        const html = `
            <div class="news-item" style="margin-bottom: 20px;">
                <h2 style="color: #667eea; margin-bottom: 15px;">${data.title}</h2>
                <div class="meta" style="margin-bottom: 15px;">
                    <strong>Автор:</strong> ${data.author}<br>
                    <strong>Категория:</strong> ${data.category ? data.category.title : 'N/A'}<br>
                    <strong>Опубликовано:</strong> ${publishedDate}
                </div>
                <div style="margin: 15px 0; padding: 15px; background: #f8f9fa; border-radius: 6px;">
                    <strong>Содержание:</strong><br>
                    ${data.content || 'Нет содержимого'}
                </div>
                <div class="tags">${tagsHtml}</div>
            </div>
            <pre>${JSON.stringify(data, null, 2)}</pre>
        `;
        resultDiv.innerHTML = html;
    });
}

// Get all categories
async function getAllCategories() {
    jsonRpcCall('news.Categories', {}, 'categoriesResult', (elementId, data) => {
        const resultDiv = document.getElementById(elementId);
        resultDiv.className = 'result';
        
        if (!data || !Array.isArray(data) || data.length === 0) {
            resultDiv.innerHTML = '<div class="empty">Категории не найдены</div>';
            return;
        }

        let html = `<div style="margin-bottom: 15px;"><strong>Найдено категорий: ${data.length}</strong></div>`;
        data.forEach(category => {
            html += `
                <div class="category-item">
                    <strong>${category.title}</strong> 
                    (ID: ${category.categoryId})
                </div>
            `;
        });
        
        html += `<pre style="margin-top: 20px;">${JSON.stringify(data, null, 2)}</pre>`;
        resultDiv.innerHTML = html;
    });
}

// Get all tags
async function getAllTags() {
    jsonRpcCall('news.Tags', {}, 'tagsResult', (elementId, data) => {
        const resultDiv = document.getElementById(elementId);
        resultDiv.className = 'result';
        
        if (!data || !Array.isArray(data) || data.length === 0) {
            resultDiv.innerHTML = '<div class="empty">Теги не найдены</div>';
            return;
        }

        let html = `<div style="margin-bottom: 15px;"><strong>Найдено тегов: ${data.length}</strong></div>`;
        data.forEach(tag => {
            html += `
                <div class="tag-item">
                    <span class="tag">${tag.title}</span> 
                    (ID: ${tag.tagId}, Статус: ${tag.statusId})
                </div>
            `;
        });
        
        html += `<pre style="margin-top: 20px;">${JSON.stringify(data, null, 2)}</pre>`;
        resultDiv.innerHTML = html;
    });
}

