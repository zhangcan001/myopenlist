use std::sync::{
    Arc,
    atomic::{AtomicBool, Ordering},
};

use base64::{Engine as _, engine::general_purpose::STANDARD};
use serde::{Deserialize, Serialize};
use tauri::{
    AppHandle, Emitter, EventTarget, Manager, WebviewUrl, WebviewWindowBuilder, WindowEvent,
};
use url::Url;

const AUTH_WINDOW_LABEL: &str = "media-drive-115-auth";
const AUTHORIZED_EVENT: &str = "media-drive-115-authorized";
const CLOSED_EVENT: &str = "media-drive-115-auth-closed";
const HOSTED_AUTH_URL: &str =
    "https://api.oplist.org/115cloud/requests?driver_txt=115cloud_go&server_use=true";

// The hosted endpoint first returns JSON containing the 115 authorization URL.
// Continue in the same webview so its state cookies are retained for the callback.
const HOSTED_AUTH_BOOTSTRAP: &str = r#"
(() => {
  if (location.hostname !== 'api.oplist.org' || location.pathname !== '/115cloud/requests') return;
  const continueAuthorization = () => {
    try {
      const response = JSON.parse(document.body.innerText);
      const next = new URL(response.text);
      const official115 = next.protocol === 'https:' &&
        (next.hostname === '115.com' || next.hostname.endsWith('.115.com'));
      if (official115) location.replace(next.href);
    } catch (_) {}
  };
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', continueAuthorization, { once: true });
  } else {
    continueAuthorization();
  }
})();
"#;

#[derive(Deserialize)]
struct HostedCallback {
    access_token: String,
    refresh_token: String,
    driver_txt: Option<String>,
    server_use: Option<String>,
}

#[derive(Clone, Serialize)]
struct HostedTokens {
    access_token: String,
    refresh_token: String,
}

fn is_allowed_auth_url(url: &Url) -> bool {
    if url.scheme() != "https" {
        return false;
    }
    matches!(url.host_str(), Some("api.oplist.org"))
        || url
            .host_str()
            .is_some_and(|host| host == "115.com" || host.ends_with(".115.com"))
}

fn callback_tokens(url: &Url) -> Option<HostedTokens> {
    if url.host_str() != Some("api.oplist.org") || url.path() != "/" {
        return None;
    }
    let fragment = url.fragment()?;
    if fragment.is_empty() || fragment.len() > 32_768 {
        return None;
    }
    let callback: HostedCallback = serde_json::from_slice(&STANDARD.decode(fragment).ok()?).ok()?;
    if callback.driver_txt.as_deref() != Some("115cloud_go")
        || callback.server_use.as_deref() != Some("true")
        || callback.access_token.trim().is_empty()
        || callback.refresh_token.trim().is_empty()
    {
        return None;
    }
    Some(HostedTokens {
        access_token: callback.access_token,
        refresh_token: callback.refresh_token,
    })
}

#[tauri::command]
pub async fn open_115_hosted_authorization(app_handle: AppHandle) -> Result<bool, String> {
    if let Some(window) = app_handle.get_webview_window(AUTH_WINDOW_LABEL) {
        window.show().map_err(|error| error.to_string())?;
        window.set_focus().map_err(|error| error.to_string())?;
        return Ok(true);
    }

    let start_url = Url::parse(HOSTED_AUTH_URL).map_err(|error| error.to_string())?;
    let completed = Arc::new(AtomicBool::new(false));
    let navigation_completed = Arc::clone(&completed);
    let navigation_app = app_handle.clone();

    let window = WebviewWindowBuilder::new(
        &app_handle,
        AUTH_WINDOW_LABEL,
        WebviewUrl::External(start_url),
    )
    .title("授权 115")
    .inner_size(900.0, 720.0)
    .min_inner_size(720.0, 560.0)
    .center()
    .incognito(true)
    .initialization_script(HOSTED_AUTH_BOOTSTRAP)
    .on_navigation(move |url| {
        if let Some(tokens) = callback_tokens(url) {
            navigation_completed.store(true, Ordering::Release);
            let _ = navigation_app.emit_to(
                EventTarget::webview_window("main"),
                AUTHORIZED_EVENT,
                tokens,
            );
            let app_to_close = navigation_app.clone();
            let _ = navigation_app.run_on_main_thread(move || {
                if let Some(window) = app_to_close.get_webview_window(AUTH_WINDOW_LABEL) {
                    let _ = window.close();
                }
            });
            return false;
        }
        is_allowed_auth_url(url)
    })
    .build()
    .map_err(|error| error.to_string())?;

    let close_app = app_handle.clone();
    window.on_window_event(move |event| {
        if matches!(event, WindowEvent::Destroyed) && !completed.load(Ordering::Acquire) {
            let _ = close_app.emit_to(EventTarget::webview_window("main"), CLOSED_EVENT, ());
        }
    });

    Ok(true)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn hosted_callback_is_decoded_only_for_openlist_115() {
        let payload = serde_json::json!({
            "access_token": "access-secret",
            "refresh_token": "refresh-secret",
            "driver_txt": "115cloud_go",
            "server_use": "true"
        });
        let fragment = STANDARD.encode(serde_json::to_vec(&payload).unwrap());
        let url = Url::parse(&format!("https://api.oplist.org/#{fragment}")).unwrap();
        let tokens = callback_tokens(&url).unwrap();

        assert_eq!(tokens.access_token, "access-secret");
        assert_eq!(tokens.refresh_token, "refresh-secret");
    }

    #[test]
    fn navigation_is_limited_to_openlist_and_115() {
        assert!(is_allowed_auth_url(
            &Url::parse("https://api.oplist.org/115cloud/requests").unwrap()
        ));
        assert!(is_allowed_auth_url(
            &Url::parse("https://passportapi.115.com/open/authorize").unwrap()
        ));
        assert!(!is_allowed_auth_url(
            &Url::parse("https://example.com/steal").unwrap()
        ));
        assert!(!is_allowed_auth_url(
            &Url::parse("http://api.oplist.org/115cloud/requests").unwrap()
        ));
    }
}
