import type { CodegenConfig } from '@graphql-codegen/cli';

const config: CodegenConfig = {
  schema: '../graphql-bff/graph/schema.graphqls',
  documents: 'src/graphql/**/*.graphql',
  generates: {
    'src/graphql/generated.ts': {
      plugins: [
        'typescript',
        'typescript-operations',
        'typescript-react-apollo',
      ],
      config: {
        withHooks: true,
        withComponent: false,
        withHOC: false,
        scalars: {
          Time: 'string',
          ID: 'string',
        },
        enumsAsTypes: true,
        avoidOptionals: false,
        dedupeFragments: true,
      },
    },
  },
};

export default config;
