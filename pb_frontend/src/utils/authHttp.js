/**
 * POST JSON to password auth endpoints; follows redirects so Set-Cookie applies.
 * Returns { ok: true } after navigation, or { ok: false, message } for display.
 */
export async function postPasswordAuth(url, body) {
  let res
  try {
    res = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json, text/plain',
      },
      body: JSON.stringify(body),
      credentials: 'same-origin',
      redirect: 'follow',
    })
  } catch {
    return {
      ok: false,
      message: 'Не удалось связаться с сервером. Проверьте соединение.',
    }
  }

  if (res.ok && res.redirected) {
    window.location.href = res.url
    return { ok: true }
  }

  if (res.ok && !res.redirected) {
    window.location.href = '/main'
    return { ok: true }
  }

  const raw = (await res.text()).trim()
  return {
    ok: false,
    message: mapAuthErrorMessage(res.status, raw),
  }
}

const BODY_TO_RU = {
  'Invalid credentials': 'Неверный логин или пароль.',
  'Could not register': 'Не удалось зарегистрироваться. Возможно, логин уже занят.',
  'Missing fields': 'Заполните все поля.',
  'Internal server error': 'Ошибка сервера. Попробуйте позже.',
  'Method not allowed': 'Неверный метод запроса.',
}

function mapAuthErrorMessage(status, bodyText) {
  if (BODY_TO_RU[bodyText]) {
    return BODY_TO_RU[bodyText]
  }
  if (status === 401) {
    return 'Неверный логин или пароль.'
  }
  if (status === 400) {
    return BODY_TO_RU[bodyText] || 'Проверьте введённые данные.'
  }
  if (status === 500 || status === 502 || status === 503) {
    return 'Сервер временно недоступен. Попробуйте позже.'
  }
  if (status === 0) {
    return 'Нет ответа от сервера.'
  }
  return 'Что-то пошло не так. Попробуйте снова.'
}
