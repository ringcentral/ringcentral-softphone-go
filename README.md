# RingCentral Softphone SDK for GoLang

## Usage

Please refer to the [examples](./examples)


## Debugging

We use environment variables to enable logging for debugging purpose.

- To debug WebRTC: `PION_LOG_DEBUG=all`
- To set Softphone log levels `RINGCENTRAL_SOFTPHONE_DEBUG=all`


## Known limitations

Current the library only accept inbound data: inbound call and inbound audio. There is currently no way to make outbound call or send audio outbound.

Outbound features are in the todo list but currently there is no timeline.
