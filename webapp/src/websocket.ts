// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {WebSocketMessage} from '@mattermost/client';

import {PostUpdateWebsocketMessage} from './components/llmbot_post/llmbot_post';

type WebsocketListener = (msg: WebSocketMessage<PostUpdateWebsocketMessage>) => void
type WebsocketListenerObject = {
    postID: string;
    listenerID: string;
    listener: WebsocketListener;
}
type WebsocketListeners = WebsocketListenerObject[]

export type ToolPermission = 'auto-approve' | 'ask';

export type ToolPermissionWebsocketMessage = {
    user_id: string;
    root_post_id: string;
    tool_name: string;
    permission: ToolPermission;
}

type ToolPermissionListener = (msg: WebSocketMessage<ToolPermissionWebsocketMessage>) => void
type ToolPermissionListenerObject = {
    rootPostID: string;
    listenerID: string;
    listener: ToolPermissionListener;
}
type ToolPermissionListeners = ToolPermissionListenerObject[]

export default class PostEventListener {
    postUpdateWebsocketListeners: WebsocketListeners = [];
    toolPermissionWebsocketListeners: ToolPermissionListeners = [];

    public registerPostUpdateListener = (postID: string, listenerID: string, listener: WebsocketListener) => {
        this.postUpdateWebsocketListeners.push({postID, listenerID, listener});
    };

    public unregisterPostUpdateListener = (postID: string, listenerID: string) => {
        this.postUpdateWebsocketListeners = this.postUpdateWebsocketListeners.filter((listenerObject) => {
            const isSamePostID = listenerObject.postID === postID;
            const isSameListenerID = listenerObject.listenerID === listenerID;
            return !(isSamePostID && isSameListenerID);
        });
    };

    public handlePostUpdateWebsockets = (msg: WebSocketMessage<PostUpdateWebsocketMessage>) => {
        const postID = msg.data.post_id;
        this.postUpdateWebsocketListeners.forEach((listenerObject) => {
            if (listenerObject.postID === postID) {
                listenerObject.listener(msg);
            }
        });
    };

    public registerToolPermissionListener = (rootPostID: string, listenerID: string, listener: ToolPermissionListener) => {
        this.toolPermissionWebsocketListeners.push({rootPostID, listenerID, listener});
    };

    public unregisterToolPermissionListener = (rootPostID: string, listenerID: string) => {
        this.toolPermissionWebsocketListeners = this.toolPermissionWebsocketListeners.filter((listenerObject) => {
            const isSameRootPostID = listenerObject.rootPostID === rootPostID;
            const isSameListenerID = listenerObject.listenerID === listenerID;
            return !(isSameRootPostID && isSameListenerID);
        });
    };

    public handleToolPermissionWebsockets = (msg: WebSocketMessage<ToolPermissionWebsocketMessage>) => {
        const rootPostID = msg.data.root_post_id;
        this.toolPermissionWebsocketListeners.forEach((listenerObject) => {
            if (listenerObject.rootPostID === rootPostID) {
                listenerObject.listener(msg);
            }
        });
    };
}
