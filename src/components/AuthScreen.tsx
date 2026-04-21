import React, { useState } from 'react';
import { IconMail, IconLock, IconUser, IconEye, IconEyeOff } from './icons';
import { t } from '../utils/i18n';

interface AuthScreenProps {
  mode: 'login' | 'register';
  onSetMode: (m: 'login' | 'register') => void;
  email: string;
  password: string;
  username: string;
  onEmail: (v: string) => void;
  onPassword: (v: string) => void;
  onUsername: (v: string) => void;
  error: string | null;
  busy: boolean;
  onLogin: (e: React.FormEvent) => void;
  onRegister: (e: React.FormEvent) => void;
}

const AuthScreen: React.FC<AuthScreenProps> = ({
  mode,
  onSetMode,
  email,
  password,
  username,
  onEmail,
  onPassword,
  onUsername,
  error,
  busy,
  onLogin,
  onRegister,
}) => {
  const [showPassword, setShowPassword] = useState(false);
  const isLogin = mode === 'login';
  const USERNAME_MAX = 32;

  const sanitizeUsername = (v: string) => {
    // Letters/digits/underscore/dot, no spaces; keep it simple and predictable.
    const cleaned = v.replace(/[^\p{L}\p{N}._]/gu, '');
    return cleaned.slice(0, USERNAME_MAX);
  };

  return (
    <div className="sf-auth-overlay">
      <div className="sf-auth-modal" role="dialog" aria-labelledby="sf-auth-title">
        <div className="sf-auth-modal-glow sf-auth-modal-glow--tr" aria-hidden />
        <div className="sf-auth-modal-glow sf-auth-modal-glow--bl" aria-hidden />

        <div className="sf-auth-head">
          <div className="sf-auth-head-text">
            <h1 id="sf-auth-title" className="sf-auth-title">
              {isLogin ? t('auth.welcomeBack') : t('auth.createAccount')}
            </h1>
            <p className="sf-auth-subtitle">
              {isLogin ? t('auth.subtitleLogin') : t('auth.subtitleRegister')}
            </p>
          </div>
        </div>

        <form
          className="sf-auth-form"
          onSubmit={isLogin ? onLogin : onRegister}
          noValidate
        >
          {!isLogin && (
            <div className="sf-auth-field">
              <label className="sf-auth-label">{t('auth.username')}</label>
              <div className="sf-auth-input-wrap">
                <span className="sf-auth-input-icon" aria-hidden>
                  <IconUser width={20} height={20} />
                </span>
                <input
                  type="text"
                  value={username}
                  onChange={(e) => onUsername(sanitizeUsername(e.target.value))}
                  placeholder={t('auth.usernamePlaceholder')}
                  className="sf-auth-input"
                  required
                  autoComplete="username"
                  maxLength={USERNAME_MAX}
                />
              </div>
              <p className="sf-input-hint">{t('auth.usernameHint')}</p>
            </div>
          )}

          <div className="sf-auth-field">
            <label className="sf-auth-label">{t('auth.email')}</label>
            <div className="sf-auth-input-wrap">
              <span className="sf-auth-input-icon" aria-hidden>
                <IconMail width={20} height={20} />
              </span>
              <input
                type="email"
                value={email}
                onChange={(e) => onEmail(e.target.value)}
                placeholder={t('auth.emailPlaceholder')}
                className="sf-auth-input"
                required
                autoComplete="email"
              />
            </div>
          </div>

          <div className="sf-auth-field">
            <label className="sf-auth-label">{t('auth.password')}</label>
            <div className="sf-auth-input-wrap">
              <span className="sf-auth-input-icon" aria-hidden>
                <IconLock width={20} height={20} />
              </span>
              <input
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => onPassword(e.target.value)}
                placeholder={t('auth.passwordPlaceholder')}
                className="sf-auth-input sf-auth-input--with-toggle"
                required
                minLength={isLogin ? undefined : 8}
                autoComplete={isLogin ? 'current-password' : 'new-password'}
              />
              <button
                type="button"
                className="sf-auth-toggle-pw"
                onClick={() => setShowPassword((s) => !s)}
                aria-label={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
              >
                {showPassword ? <IconEyeOff width={20} height={20} /> : <IconEye width={20} height={20} />}
              </button>
            </div>
            {!isLogin ? <p className="sf-input-hint">{t('auth.passwordHint')}</p> : null}
          </div>

          {isLogin && (
            <div className="sf-auth-row">
              <label className="sf-auth-remember">
                <input type="checkbox" className="sf-auth-checkbox" />
                <span>{t('auth.rememberMe')}</span>
              </label>
            </div>
          )}

          {error && <p className="sf-auth-error">{error}</p>}

          <button type="submit" className="sf-auth-submit" disabled={busy}>
            {busy ? '…' : isLogin ? t('auth.login') : t('auth.create')}
          </button>
        </form>

        <div className="sf-auth-footer">
          <p className="sf-auth-switch">
            {isLogin ? t('auth.haveNoAccount') : t('auth.haveAccount')}
            <button
              type="button"
              className="sf-auth-switch-btn"
              onClick={() => {
                onSetMode(isLogin ? 'register' : 'login');
              }}
            >
              {isLogin ? t('auth.signUp') : t('auth.login')}
            </button>
          </p>
        </div>
      </div>
    </div>
  );
};

export default AuthScreen;
