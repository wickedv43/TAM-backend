/* Do not change, this code is generated from Golang structs */


export enum DealType {
    ADVERTISING_OFFER = 0,
    CHANNEL_OFFER = 1,
}
export enum DealStatus {
    DRAFT = 0,
    PENDING = 1,
    DISCUSSION = 2,
    IN_PROGRESS = 3,
    AWAITING_APPROVAL = 4,
    AWAITING_APPROVAL_TIME = 5,
    AWAITING_PUBLICATION = 6,
    PUBLISHED = 7,
    CANCELED = 8,
    EXPIRED = 9,
    TERMS_VIOLATED = 10,
    COMPLETED = 11,
    ERR_PUBLISH = 12,
    ERR_BAD_TARGET = 13,
}
export enum DealTargetType {
    post_1_24 = "post_1_24",
    post_2_48 = "post_2_48",
    post_3_72 = "post_3_72",
}
export enum CustomerStatus {
    PRE_REGISTRATION = 0,
    REGISTERED = 1,
}
export enum ChannelStatus {
    UNACTIVATED = 0,
    AWAITING_DATA = 1,
    ACTIVATED = 2,
    VERIFIED = 3,
}
export enum CustomerChannelRole {
    ADMIN = 0,
    OWNER = 1,
}
export enum TransactionStatus {
    PROCESSING = 0,
    DONE = 1,
}
export interface AuthRequest {
    init_data: string;
    ref_id: string;
}
export interface AuthResponse {
    token: string;
}
export interface BriefResponse {
    id: string;
    created_at: string;
    updated_at: string;
}
export interface AddChannelRequest {
    channel_username: string;
}
export interface ChannelPrices {
    post_1_24: number;
    post_2_48?: number;
    post_3_72?: number;
}
export interface BroadcastValue {
    current: number;
    previous: number;
}
export interface BroadcastPeriod {
    min_date: string;
    max_date: string;
}
export interface BroadcastStats {
    period: BroadcastPeriod;
    followers: BroadcastValue;
    views_per_post: BroadcastValue;
    shares_per_post: BroadcastValue;
    reactions_per_post: BroadcastValue;
    views_per_story: BroadcastValue;
    shares_per_story: BroadcastValue;
    reactions_per_story: BroadcastValue;
    enabled_notifications: BroadcastValue;
}
export interface PostsStats {
    date_from: string;
    date_to: string;
    count: number;
    avg_views: number;
    median_views: number;
    avg_forwards: number;
    median_forwards: number;
    avg_reactions: number;
    median_reactions: number;
}
export interface LanguageStat {
    name: string;
    total: number;
    ratio: number;
}
export interface Graph {
    data?: string;
}
export interface ChannelGraphs {
    mute?: Graph;
    growth?: Graph;
    followers?: Graph;
    top_hours?: Graph;
    languages?: Graph;
    views_by_source?: Graph;
    interactions?: Graph;
    iv_interactions?: Graph;
    new_followers_by_source?: Graph;
    reactions_by_emotion?: Graph;
    story_interactions?: Graph;
    story_reactions_by_emotion?: Graph;
}
export interface ChannelStats {
    graphs: ChannelGraphs;
    languages?: LanguageStat[];
    posts?: PostsStats;
    broadcast?: BroadcastStats;
}
export interface ChannelResponse {
    id: string;
    tg_id: string;
    tg_username: string;
    tg_name: string;
    tg_description: string;
    tg_picture: string;
    commentary: string;
    is_listed: boolean;
    tags: string[];
    stats_updated_at: string;
    stats?: ChannelStats;
    prices: ChannelPrices;
    subscribers: number;
    premium_subscribers: number;
    median_post_views: number;
    avg_post_views: number;
    status: number;
    main_language: string;
    restricted_from: string;
    restricted_till: string;
    restriction_reason: string;
    notifications_on: number;
    first_post_date: string;
    total_posts: number;
    created_at: string;
    updated_at: string;
}
export interface ChannelResponseWithoutStats {
    id: string;
    tg_id: string;
    tg_username: string;
    tg_name: string;
    tg_description: string;
    tg_picture: string;
    commentary: string;
    is_listed: boolean;
    tags: string[];
    stats_updated_at: string;
    prices: ChannelPrices;
    subscribers: number;
    premium_subscribers: number;
    median_post_views: number;
    avg_post_views: number;
    status: ChannelStatus;
    main_language: string;
    restricted_from: string;
    restricted_till: string;
    restriction_reason: string;
    notifications_on: number;
    first_post_date: string;
    total_posts: number;
    created_at: string;
    updated_at: string;
}
export interface GetChannelRequest {
    channel_id: string;
}
export interface GetChannelResponse {
    canEdit: boolean;
    channel: ChannelResponse;
}

export interface ProvideChannelDataRequest {
    channel_id: string;
    channel_description: string;
    tags: string[];
    language: string;
    prices: ChannelPrices;
}
export interface PageOptions {
    page: number;
    take: number;
    order: string;
}
export interface ListChannelsRequest {
    page_options: PageOptions;
    my_channels: boolean;
}
export interface ListChannelsResponse {
    data: ChannelResponseWithoutStats[];
    count: number;
}








export interface UpdateMeRequest {
    tg_username?: string;
    tg_firstname?: string;
    tg_lastname?: string;
    tg_language?: string;
    tg_picture?: string;
    address_bounceable?: string;
    address_nonbounceable?: string;
}
export interface CustomerResponse {
    id: string;
    tg_id: string;
    tg_username: string;
    tg_firstname: string;
    tg_lastname: string;
    tg_language: string;
    tg_picture: string;
    tg_is_premium: boolean;
    address_bounceable: string;
    address_nonbounceable: string;
    ton_balance: number;
    ton_balance_locked: number;
    status: CustomerStatus;
    created_at: string;
    updated_at: string;
}
export interface WithdrawRequest {
    amount: number;
    to_address: string;
}
export interface WithdrawResponse {
    tx_hash: string;
}
export interface DealMessageRequest {
    deal_id: string;
}
export interface SendDealDataRequest {
    deal_id: string;
}
export interface OfferDealRequest {
    channel_id: string;
    target_type: string;
}
export interface PreferableDatetime {
    date_from?: string;
    date_to?: string;
    time_from?: string;
    time_to?: string;
}
export interface MakeDealRequest {
    deal_id: string;
    preferable_datetime?: PreferableDatetime;
}
export interface ApproveAndProposeTimeRequest {
    deal_id: string;
    publication_time: string;
}
export interface UpdateDealRequest {
    status?: DealStatus;
    expires_at?: string;
    target: {[key: string]: any};
    publication_time?: string;
}
export interface TgMsgMediaItem {
    file_id: string;
    msg_id: number;
    type: string;
    caption_above: boolean;
    has_spoiler?: boolean;
}
export interface TgMsgData {
    message_text?: string;
    message_entities?: any[];
    message_images?: TgMsgMediaItem[];
    message_caption_above?: boolean;
    message_album_id?: string;
    message_preview_ids?: any[];
    message_deal_id?: string;
}
export interface DealTarget {
    brief_data?: TgMsgData;
    post_data?: TgMsgData;
    preferable_datetime?: PreferableDatetime;
    story_data?: {[key: string]: any};
    repost_data?: {[key: string]: any};
}
export interface DealResponse {
    id: string;
    channel_id: string;
    advertiser_customer_id: string;
    channel_manager_id: string;
    type: DealType;
    status: DealStatus;
    status_updated_at: string;
    expires_at: string;
    target_type: string;
    target: DealTarget;
    publication_time?: string;
    top_deadline?: string;
    common_deadline?: string;
    ton_price: number;
    created_at: string;
    updated_at: string;
}
export interface DealRequest {
    deal_id: string;
}
export interface GetMyDealsResponse {
    advertiser_deals: DealResponse[];
    manager_deals: DealResponse[];
}
export interface ListDealsRequest {
    page_options: PageOptions;
}
export interface ListDealsResponse {
    data: DealResponse[];
    count: number;
}
export interface DealWithChannelResponse {
    deal: DealResponse;
    channel: ChannelResponseWithoutStats;
}
export interface ListDealsWithChannelResponse {
    data: DealWithChannelResponse[];
    count: number;
}