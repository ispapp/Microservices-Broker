
const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const proto = require('./base');
const PROTO_PATH = './base/base.proto';
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true
});
const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
const BrokerService = protoDescriptor.base.proto.Broker;

class ServiceClient {
    constructor(serviceName, serviceUrl) {
        this.serviceName = serviceName;
        this.client = new BrokerService(serviceUrl, grpc.credentials.createInsecure());
    }

    ping(callback) {
        this.client.Ping({ from: this.serviceName }, callback);
    }

    sendMessage(to, data, type, queue, callback) {
        const message = {
            from: this.serviceName,
            to,
            data: Buffer.from(data),
            type,
            queue
        };
        this.client.Send(message, callback);
    }

    receiveMessages(callback) {
        const call = this.client.Receive({ from: this.serviceName });
        call.on('data', callback);
        call.on('end', () => console.log('Stream ended'));
        call.on('error', (e) => console.error('Stream error:', e));
    }

    cleanup(callback) {
        this.client.Cleanup({ from: this.serviceName }, callback);
    }
}

module.exports = ServiceClient;