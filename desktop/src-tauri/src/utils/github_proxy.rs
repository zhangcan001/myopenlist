pub fn apply_github_proxy(
    url: &str,
    gh_proxy: &Option<String>,
    gh_proxy_api: &Option<bool>,
) -> String {
    if let Some(proxy) = gh_proxy
        && !proxy.is_empty()
        && should_proxy_url(url, gh_proxy_api)
    {
        let proxy_clean = proxy.trim_end_matches('/');
        return format!("{proxy_clean}/{url}");
    }
    url.to_string()
}

fn should_proxy_url(url: &str, gh_proxy_api: &Option<bool>) -> bool {
    if url.starts_with("https://github.com/") {
        return true;
    }

    if url.starts_with("https://api.github.com/") {
        return gh_proxy_api.unwrap_or(false);
    }

    false
}
