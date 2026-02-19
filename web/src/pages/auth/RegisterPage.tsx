import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Link, useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import { useAuth } from '../../hooks/useAuth';

const registerSchema = z.object({
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
  name: z.string().min(1, 'Name is required'),
  orgName: z.string().min(1, 'Organization name is required'),
});

type RegisterForm = z.infer<typeof registerSchema>;

export function RegisterPage() {
  const { register: authRegister } = useAuth();
  const navigate = useNavigate();
  const [error, setError] = useState('');

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<RegisterForm>({
    resolver: zodResolver(registerSchema),
  });

  const onSubmit = async (data: RegisterForm) => {
    setError('');
    try {
      await authRegister(data.email, data.password, data.name, data.orgName);
      navigate({ to: '/' });
    } catch (err: unknown) {
      if (
        err &&
        typeof err === 'object' &&
        'response' in err &&
        (err as { response?: { status?: number } }).response?.status === 409
      ) {
        setError('Email or organization name already taken');
      } else {
        setError('Registration failed. Please try again.');
      }
    }
  };

  return (
    <div className="auth-page">
      <div className="auth-card">
        <h1>Create your account</h1>
        {error && <div className="auth-error">{error}</div>}
        <form onSubmit={handleSubmit(onSubmit)}>
          <div className="form-group">
            <label htmlFor="name">Name</label>
            <input id="name" type="text" {...register('name')} />
            {errors.name && (
              <span className="field-error">{errors.name.message}</span>
            )}
          </div>
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
              autoComplete="new-password"
            />
            {errors.password && (
              <span className="field-error">{errors.password.message}</span>
            )}
          </div>
          <div className="form-group">
            <label htmlFor="orgName">Organization name</label>
            <input id="orgName" type="text" {...register('orgName')} />
            {errors.orgName && (
              <span className="field-error">{errors.orgName.message}</span>
            )}
          </div>
          <button type="submit" disabled={isSubmitting} className="btn-primary">
            {isSubmitting ? 'Creating account...' : 'Create account'}
          </button>
        </form>
        <p className="auth-link">
          Already have an account? <Link to="/auth/login">Sign in</Link>
        </p>
      </div>
    </div>
  );
}
