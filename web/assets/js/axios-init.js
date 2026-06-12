axios.defaults.headers.post['Content-Type'] = 'application/x-www-form-urlencoded; charset=UTF-8';
axios.defaults.headers.common['X-Requested-With'] = 'XMLHttpRequest';

axios.interceptors.request.use(
    config => {
        if (window.XUI_CSRF_TOKEN && ['post', 'put', 'patch', 'delete'].includes((config.method || '').toLowerCase())) {
            config.headers['X-CSRF-Token'] = window.XUI_CSRF_TOKEN;
        }
        config.data = Qs.stringify(config.data, {
            arrayFormat: 'repeat'
        });
        return config;
    },
    error => Promise.reject(error)
);
