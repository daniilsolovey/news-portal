const API_BASE_URL = 'http://localhost:3000/api/v1';

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

// Helper function to handle API errors
async function handleApiCall(url, elementId, customFormatter = null) {
    showLoading(elementId);
    try {
        const response = await fetch(url);
        const data = await response.json();
        
        if (!response.ok) {
            displayResult(elementId, { error: data.error || 'Ошибка запроса', status: response.status }, true);
            return;
        }

        if (customFormatter) {
            customFormatter(elementId, data);
        } else {
            displayResult(elementId, data);
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

    let url = `${API_BASE_URL}/news?page=${page}&pageSize=${pageSize}`;
    if (tagId) url += `&tagId=${tagId}`;
    if (categoryId) url += `&categoryId=${categoryId}`;

    handleApiCall(url, 'allNewsResult', (elementId, data) => {
        const resultDiv = document.getElementById(elementId);
        resultDiv.className = 'result';
        
        if (!data || data.length === 0) {
            resultDiv.innerHTML = '<div class="empty">Новости не найдены</div>';
            return;
        }

        let html = `<div style="margin-bottom: 15px;"><strong>Найдено новостей: ${data.length}</strong></div>`;
        data.forEach(news => {
            const publishedDate = new Date(news.publishedAt).toLocaleString('ru-RU');
            const tagsHtml = news.tags.map(tag => `<span class="tag">${tag.title}</span>`).join('');
            
            html += `
                <div class="news-item">
                    <h3>${news.title}</h3>
                    <div class="meta">
                        <strong>Автор:</strong> ${news.author} | 
                        <strong>Категория:</strong> ${news.category.title} | 
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

    let url = `${API_BASE_URL}/news/count`;
    if (tagId) url += `?tagId=${tagId}`;
    if (categoryId) {
        url += tagId ? `&categoryId=${categoryId}` : `?categoryId=${categoryId}`;
    }

    handleApiCall(url, 'countResult', (elementId, data) => {
        const resultDiv = document.getElementById(elementId);
        resultDiv.className = 'result';
        resultDiv.innerHTML = `
            <div style="text-align: center; padding: 20px;">
                <h2 style="color: #667eea; font-size: 2.5em; margin-bottom: 10px;">${data}</h2>
                <p style="color: #666;">новостей найдено</p>
            </div>
            <pre>${JSON.stringify(data, null, 2)}</pre>
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

    const url = `${API_BASE_URL}/news/${newsId}`;
    
    handleApiCall(url, 'newsByIdResult', (elementId, data) => {
        const resultDiv = document.getElementById(elementId);
        resultDiv.className = 'result';
        
        const publishedDate = new Date(data.publishedAt).toLocaleString('ru-RU');
        const updatedDate = data.updatedAt ? new Date(data.updatedAt).toLocaleString('ru-RU') : 'Не обновлялась';
        const tagsHtml = data.tags.map(tag => `<span class="tag">${tag.title}</span>`).join('');
        
        const html = `
            <div class="news-item" style="margin-bottom: 20px;">
                <h2 style="color: #667eea; margin-bottom: 15px;">${data.title}</h2>
                <div class="meta" style="margin-bottom: 15px;">
                    <strong>Автор:</strong> ${data.author}<br>
                    <strong>Категория:</strong> ${data.category.title}<br>
                    <strong>Опубликовано:</strong> ${publishedDate}<br>
                    <strong>Обновлено:</strong> ${updatedDate}
                </div>
                <div style="margin: 15px 0; padding: 15px; background: #f8f9fa; border-radius: 6px;">
                    <strong>Содержание:</strong><br>
                    ${data.content}
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
    const url = `${API_BASE_URL}/categories`;
    
    handleApiCall(url, 'categoriesResult', (elementId, data) => {
        const resultDiv = document.getElementById(elementId);
        resultDiv.className = 'result';
        
        if (!data || data.length === 0) {
            resultDiv.innerHTML = '<div class="empty">Категории не найдены</div>';
            return;
        }

        let html = `<div style="margin-bottom: 15px;"><strong>Найдено категорий: ${data.length}</strong></div>`;
        data.forEach(category => {
            html += `
                <div class="category-item">
                    <strong>${category.title}</strong> 
                    (ID: ${category.categoryId}, Порядок: ${category.orderNumber}, Статус: ${category.statusId})
                </div>
            `;
        });
        
        html += `<pre style="margin-top: 20px;">${JSON.stringify(data, null, 2)}</pre>`;
        resultDiv.innerHTML = html;
    });
}

// Get all tags
async function getAllTags() {
    const url = `${API_BASE_URL}/tags`;
    
    handleApiCall(url, 'tagsResult', (elementId, data) => {
        const resultDiv = document.getElementById(elementId);
        resultDiv.className = 'result';
        
        if (!data || data.length === 0) {
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

