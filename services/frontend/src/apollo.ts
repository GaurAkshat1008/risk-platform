import {
  ApolloClient,
  InMemoryCache,
  createHttpLink,
  from,
} from '@apollo/client';
import { setContext } from '@apollo/client/link/context';
import { onError } from '@apollo/client/link/error';
import keycloak from './keycloak';

const httpLink = createHttpLink({
  uri: import.meta.env.VITE_GRAPHQL_URL || '/graphql',
});

const authLink = setContext(async (_, { headers }) => {
  try {
    await keycloak.updateToken(30);
  } catch {
    keycloak.login();
  }
  return {
    headers: {
      ...headers,
      authorization: keycloak.token ? `Bearer ${keycloak.token}` : '',
    },
  };
});

const errorLink = onError(({ graphQLErrors, networkError }) => {
  if (graphQLErrors) {
    for (const err of graphQLErrors) {
      console.error('[GraphQL Error]', err.message, err.path);
    }
  }
  if (networkError) {
    console.error('[Network Error]', networkError);
  }
});

const client = new ApolloClient({
  link: from([errorLink, authLink, httpLink]),
  cache: new InMemoryCache({
    typePolicies: {
      Query: {
        fields: {
          decisions: { keyArgs: ['tenantId', 'outcomeFilter'] },
          cases: { keyArgs: ['tenantId', 'status', 'assigneeId'] },
          auditTrail: { keyArgs: ['query'] },
          queryLogs: { keyArgs: ['query'] },
        },
      },
    },
  }),
  defaultOptions: {
    watchQuery: { fetchPolicy: 'cache-and-network' },
    query: { fetchPolicy: 'network-only' },
  },
});

export default client;
