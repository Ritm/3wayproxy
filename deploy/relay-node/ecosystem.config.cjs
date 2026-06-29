module.exports = {
  apps: [
    {
      name: '3wayproxy-relay',
      script: 'src/server.js',
      cwd: __dirname + '/../../relay-node',
      env: {
        RELAY_SHARD_ID: '0',
        PORT: '3000',
      },
    },
  ],
};
