import { notification } from 'antd';
import type { ApolloError } from '@apollo/client';

/**
 * Extract human-readable messages from an ApolloError and show
 * an Ant Design notification.
 */
export function showGqlError(error: ApolloError) {
  const messages = error.graphQLErrors?.map((e) => e.message) ?? [];
  const msg = messages.length > 0 ? messages.join('; ') : error.networkError?.message ?? 'An unexpected error occurred';
  notification.error({ message: 'Error', description: msg, duration: 5 });
}

/** Convenience: wrap a mutation call and show errors automatically. */
export async function safeMutate<T>(
  fn: () => Promise<T>,
  errorFallback?: string,
): Promise<T | null> {
  try {
    return await fn();
  } catch (err) {
    if (err && typeof err === 'object' && 'graphQLErrors' in err) {
      showGqlError(err as ApolloError);
    } else {
      notification.error({ message: 'Error', description: errorFallback ?? String(err), duration: 5 });
    }
    return null;
  }
}
