import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Link, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import { useAuth } from '../../hooks/useAuth';

const loginSchema = z.object({
  email: z.string().email('Invalid email address'),
  password: z.string().min(1, 'Password is required'),
});

type LoginForm = z.infer<typeof loginSchema>;

export function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [error, setError] = useState('');

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
  });

  const onSubmit = async (data: LoginForm) => {
    setError('');
    try {
      await login(data.email, data.password);
      navigate({ to: '/' });
    } catch {
      setError('Invalid email or password');
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1>Sign in to Kapstan</h1>
        {error && <div className="auth-error">{error}</div>}
        <form onSubmit={handleSubmit(onSubmit)}>
          <div className="form-group">
            <label htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              {...register('email')}
              autoComplete="email"
            />
            {errors.email && (
              <span className="field-error">{errors.email.message}</span>
            )}
          </div>
          <div className="form-group">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              {...register('password')}
              autoComplete="current-password"
            />
            {errors.password && (
              <span className="field-error">{errors.password.message}</span>
            )}
          </div>
          <button type="submit" disabled={isSubmitting} className="btn-primary">
            {isSubmitting ? 'Signing in...' : 'Sign in'}
          </button>
        </form>
        <p className="auth-link">
          Don't have an account? <Link to="/auth/register">Sign up</Link>
        </p>
      </div>
    </div>
  );
}
