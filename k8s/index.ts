import * as fs from 'fs';
import * as k8s from '@pulumi/kubernetes';
import * as pulumi from '@pulumi/pulumi';
import * as hash from 'object-hash';

// Read the .env file and parse its contents into an object
function parseEnvFile(envFilePath: string): { [key: string]: string } {
  const env = fs.readFileSync(envFilePath, 'utf-8');
  return env.split('\n')
    .filter(line => line.trim() !== '' && !line.startsWith('#'))
    .map(line => line.split('='))
    .reduce((envObj, [key, value]) => {
      envObj[key] = value;
      return envObj;
    }, {} as { [key: string]: string });
}

// Instantiate a Kubernetes Provider and specify the render directory.
const out_dir = new k8s.Provider("render-yaml", {
  renderYamlToDirectory: "./rendered",
});


// Path to your .env file
const envFilePath = './booksing.env';

// Parse .env file
const configMapData = parseEnvFile(envFilePath);

const cs = hash(configMapData,
  {
    algorithm: 'md5',
    encoding: 'base64'
  }).substring(0, 6);

// Create a ConfigMap with the parsed .env file contents
const configMap = new k8s.core.v1.ConfigMap('booksing-config', {
  metadata: {
    name: `booksing-config-${cs}`,
  },
  data: configMapData,
}, { provider: out_dir });

const booksingTailscaleSvc = new k8s.core.v1.Service("booksingTailscale", {
  metadata: {
    annotations: {
      "tailscale.com/hostname": "booksing",
    },
    name: "booksing-tailscale",
  },
  spec: {
    loadBalancerClass: "tailscale",
    ports: [{
      port: 80,
    }],
    selector: {
      app: "booksing",
    },
    type: "LoadBalancer",
  },
}, { provider: out_dir });

const booksingSvc = new k8s.core.v1.Service("booksing", {
  metadata: {
    name: "booksing",
  },
  spec: {
    ports: [{
      port: 80,
      protocol: "TCP",
      targetPort: 7133,
    }],
    selector: {
      app: "booksing",
    },
    type: "ClusterIP",
  },
}, { provider: out_dir });


const booksingDeployment = new k8s.apps.v1.Deployment("booksing", {
  metadata: {
    name: "booksing",
  },
  spec: {
    replicas: 1,
    selector: {
      matchLabels: {
        app: "booksing",
      },
    },
    strategy: {
      type: "Recreate",
    },
    template: {
      metadata: {
        labels: {
          app: "booksing",
        },
      },
      spec: {
        containers: [{
          envFrom: [{
            configMapRef: {
              name: configMap.metadata.name,
            },
          }],
          image: "moon.goat-gecko.ts.net/gnur/booksing",
          name: "booksing",
          ports: [{
            containerPort: 7133,
          }],
          resources: {
            limits: {
              cpu: "4",
              memory: "2048Mi",
            },
            requests: {
              cpu: "1",
              memory: "512Mi",
            },
          },
          volumeMounts: [{
            mountPath: "/data",
            name: "data",
          }],
        }],
        volumes: [{
          hostPath: {
            path: "/var/lib/booksing",
          },
          name: "data",
        }],
      },
    },
  },
}, { provider: out_dir });
