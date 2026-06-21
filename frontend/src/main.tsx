import React, { FormEvent, useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  ArrowLeft,
  Bot,
  Bookmark,
  Camera,
  ChevronRight,
  Clock3,
  CreditCard,
  Edit3,
  Home,
  LogOut,
  MessageCircle,
  Plus,
  RefreshCw,
  Search,
  Send,
  ShoppingBag,
  UserRound
} from "lucide-react";
import "./styles.css";

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

type User = {
  id: number;
  name: string;
  email: string;
  profile_image_url: string;
  bio: string;
};

type PublicProfile = {
  id: number;
  name: string;
  profile_image_url: string;
  bio: string;
};

type Listing = {
  id: number;
  seller_id: number;
  seller_name: string;
  title: string;
  description: string;
  price: number;
  image_url: string;
  image_urls: string[];
  status: "available" | "sold" | "cancelled";
  buyer_id?: number;
};

type Conversation = {
  id: number;
  listing_id: number;
  buyer_id: number;
  seller_id: number;
  title: string;
  other_user_id: number;
  other_user_name: string;
  updated_at: string;
};

type Message = {
  id: number;
  conversation_id: number;
  sender_id: number;
  body: string;
  attachment_url: string;
  created_at: string;
};

type PurchaseResult = {
  listing: Listing;
  conversation: Conversation;
  message: Message;
};

type PurchaseNotification = {
  id: number;
  listing_id: number;
  conversation_id: number;
  buyer_id: number;
  buyer_name: string;
  title: string;
  created_at: string;
};

type Tab = "home" | "sell" | "messages" | "history";
type CheckoutMode = "checkout" | "address" | "dropoff" | "payment" | "confirm" | null;

type ShippingAddress = {
  lastName: string;
  firstName: string;
  lastKana: string;
  firstKana: string;
  postalCode: string;
  prefecture: string;
  city: string;
  block: string;
  building: string;
  phone: string;
};

const emptyAddress: ShippingAddress = {
  lastName: "",
  firstName: "",
  lastKana: "",
  firstKana: "",
  postalCode: "",
  prefecture: "",
  city: "",
  block: "",
  building: "",
  phone: ""
};

const dropoffOptions = ["置き配を利用しない", "玄関前", "宅配ボックス", "メーターボックス", "物置", "車庫"];
const paymentOptions = ["VISA（クレジット）", "d払い（ドコモ）", "ソフトバンクまとめて支払い", "コンビニ/ATM払い"];
const conditionOptions = ["新品・未使用", "未使用に近い", "目立った傷や汚れなし", "やや傷や汚れあり", "傷や汚れあり", "全体的に状態が悪い"];

function App() {
  const [token, setToken] = useState("");
  const [user, setUser] = useState<User | null>(null);
  const [activeTab, setActiveTab] = useState<Tab>("home");
  const [listings, setListings] = useState<Listing[]>([]);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [messages, setMessages] = useState<Message[]>([]);
  const [purchaseNotifications, setPurchaseNotifications] = useState<PurchaseNotification[]>([]);
  const [activeConversation, setActiveConversation] = useState<number | null>(null);
  const [bookmarks, setBookmarks] = useState<number[]>([]);
  const [searchDraft, setSearchDraft] = useState("");
  const [appliedQuery, setAppliedQuery] = useState("");
  const [selectedListingID, setSelectedListingID] = useState<number | null>(null);
  const [selectedImageIndex, setSelectedImageIndex] = useState(0);
  const [detailSource, setDetailSource] = useState<"home" | "history">("home");
  const [profileMode, setProfileMode] = useState<"view" | "edit" | null>(null);
  const [profileUser, setProfileUser] = useState<PublicProfile | null>(null);
  const [editName, setEditName] = useState("");
  const [editBio, setEditBio] = useState("");
  const [editImageURL, setEditImageURL] = useState("");
  const [checkoutMode, setCheckoutMode] = useState<CheckoutMode>(null);
  const [checkoutListingID, setCheckoutListingID] = useState<number | null>(null);
  const [shippingAddress, setShippingAddress] = useState<ShippingAddress>(emptyAddress);
  const [addressDraft, setAddressDraft] = useState<ShippingAddress>(emptyAddress);
  const [dropoffLocation, setDropoffLocation] = useState("");
  const [dropoffDraft, setDropoffDraft] = useState("");
  const [paymentMethod, setPaymentMethod] = useState("");
  const [paymentDraft, setPaymentDraft] = useState("");
  const [purchaseComplete, setPurchaseComplete] = useState(false);
  const [notice, setNotice] = useState("");
  const [authMode, setAuthMode] = useState<"login" | "register">("register");

  const headers = useMemo(
    () => ({
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    }),
    [token]
  );

  const bookmarkKey = user ? `campus-market-bookmarks-${user.id}` : "";
  const visibleListings = listings.filter((item) => item.status !== "cancelled");
  const otherListings = visibleListings.filter((item) => item.seller_id !== user?.id);
  const filteredListings = otherListings.filter((item) => {
    const text = `${item.title} ${item.description} ${item.seller_name}`.toLowerCase();
    return text.includes(appliedQuery.trim().toLowerCase());
  });
  const selectedListing = visibleListings.find((item) => item.id === selectedListingID) ?? null;
  const checkoutListing = listings.find((item) => item.id === checkoutListingID) ?? selectedListing;
  const bookmarkedListings = visibleListings.filter((item) => bookmarks.includes(item.id));
  const sellingListings = visibleListings.filter((item) => item.seller_id === user?.id && item.status === "available");
  const soldListings = visibleListings.filter((item) => item.seller_id === user?.id && item.status === "sold");
  const purchasedListings = visibleListings.filter((item) => item.buyer_id === user?.id);
  const activeConversationDetail = conversations.find((conversation) => conversation.id === activeConversation);

  async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const res = await fetch(`${API_BASE}${path}`, {
      ...options,
      headers: { ...headers, ...options.headers }
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error ?? "request failed");
    return data as T;
  }

  async function loadListings() {
    const data = await request<Listing[]>("/api/listings");
    setListings(data ?? []);
  }

  async function loadConversations() {
    const data = await request<Conversation[]>("/api/conversations");
    setConversations(data ?? []);
  }

  async function loadPurchaseNotifications() {
    const data = await request<PurchaseNotification[]>("/api/notifications/purchases");
    setPurchaseNotifications(data ?? []);
  }

  async function refreshAppData() {
    await loadListings();
    if (user) {
      await loadConversations();
      await loadPurchaseNotifications();
    }
  }

  useEffect(() => {
    if (!user) return;
    loadListings().catch(() => setNotice("商品一覧を読み込めませんでした。APIサーバーの起動状況を確認してください。"));
    loadConversations().catch(() => setNotice("DM一覧を読み込めませんでした。"));
    loadPurchaseNotifications().catch(() => setNotice("購入通知を読み込めませんでした。"));
    const saved = localStorage.getItem(`campus-market-bookmarks-${user.id}`);
    setBookmarks(saved ? JSON.parse(saved) : []);
  }, [user?.id]);

  useEffect(() => {
    if (!bookmarkKey) return;
    localStorage.setItem(bookmarkKey, JSON.stringify(bookmarks));
  }, [bookmarkKey, bookmarks]);

  async function handleAuth(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setNotice("");
    const form = new FormData(event.currentTarget);
    const path = authMode === "register" ? "/api/auth/register" : "/api/auth/login";
    const payload =
      authMode === "register"
        ? {
            name: String(form.get("name") ?? ""),
            email: String(form.get("email") ?? ""),
            password: String(form.get("password") ?? "")
          }
        : {
            email: String(form.get("email") ?? ""),
            password: String(form.get("password") ?? "")
          };

    try {
      const data = await request<{ user: User; token: string }>(path, {
        method: "POST",
        body: JSON.stringify(payload)
      });
      setToken(data.token);
      setUser(data.user);
      setActiveTab("home");
      setNotice(`${data.user.name} としてログインしました`);
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "認証に失敗しました");
    }
  }

  function logout() {
    setToken("");
    setUser(null);
    setActiveTab("home");
    setListings([]);
    setConversations([]);
    setMessages([]);
    setPurchaseNotifications([]);
    setActiveConversation(null);
    setBookmarks([]);
    setSearchDraft("");
    setAppliedQuery("");
    setSelectedListingID(null);
    setDetailSource("home");
    setProfileMode(null);
    setProfileUser(null);
    setCheckoutMode(null);
    setCheckoutListingID(null);
    setPurchaseComplete(false);
    setNotice("");
    setAuthMode("login");
  }

  function switchTab(tab: Tab) {
    setActiveTab(tab);
    if (tab === "home") setSelectedListingID(null);
    setDetailSource(tab === "history" ? "history" : "home");
    setProfileMode(null);
    setProfileUser(null);
    setCheckoutMode(null);
    setCheckoutListingID(null);
    setPurchaseComplete(false);
  }

  function openOwnProfile() {
    if (!user) return;
    setProfileUser({
      id: user.id,
      name: user.name,
      profile_image_url: user.profile_image_url,
      bio: user.bio
    });
    setProfileMode("view");
  }

  function openProfileEditor() {
    if (!user) return;
    setEditName(user.name);
    setEditBio(user.bio ?? "");
    setEditImageURL(user.profile_image_url ?? "");
    setProfileMode("edit");
  }

  async function openPublicProfile(userID: number) {
    setNotice("");
    try {
      const data = await request<PublicProfile>(`/api/users/${userID}`);
      setProfileUser(data);
      setProfileMode("view");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "プロフィールを読み込めませんでした");
    }
  }

  function toggleBookmark(listingID: number) {
    setBookmarks((current) => (current.includes(listingID) ? current.filter((id) => id !== listingID) : [...current, listingID]));
  }

  function openListingDetail(item: Listing, source: "home" | "history") {
    setSelectedListingID(item.id);
    setSelectedImageIndex(0);
    setDetailSource(source);
  }

  function listingImages(item: Listing) {
    return item.image_urls?.length ? item.image_urls : item.image_url ? [item.image_url] : [];
  }

  function fileToDataURL(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(String(reader.result ?? ""));
      reader.onerror = () => reject(reader.error);
      reader.readAsDataURL(file);
    });
  }

  async function imageFileToDataURL(file: File): Promise<string> {
    if (!file.type.startsWith("image/")) {
      throw new Error("画像ファイルを選択してください。");
    }
    if (!["image/jpeg", "image/png", "image/webp"].includes(file.type)) {
      throw new Error("対応画像はJPEG、PNG、WebPです。iPhoneのHEIC画像はJPEGに変換してから選んでください。");
    }
    const source = await createImageBitmap(file);
    const maxSide = 900;
    const scale = Math.min(1, maxSide / Math.max(source.width, source.height));
    const width = Math.max(1, Math.round(source.width * scale));
    const height = Math.max(1, Math.round(source.height * scale));
    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const context = canvas.getContext("2d");
    if (!context) {
      throw new Error("画像を処理できませんでした。");
    }
    context.drawImage(source, 0, 0, width, height);
    source.close();
    return canvas.toDataURL("image/jpeg", 0.72);
  }

  async function createListing(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formElement = event.currentTarget;
    setNotice("");
    const form = new FormData(formElement);
    const images = form.getAll("images").filter((image): image is File => image instanceof File && image.size > 0);
    try {
      if (images.length > 5) {
        setNotice("画像は5枚まで選択できます。");
        return;
      }
      if (images.some((image) => image.size > 12000000)) {
        setNotice("画像が大きすぎます。1枚あたり12MB以下の画像を選んでください。");
        return;
      }
      const imageURLs = await Promise.all(images.map(imageFileToDataURL));
      await request<Listing>("/api/listings", {
        method: "POST",
        body: JSON.stringify({
          title: String(form.get("title") ?? ""),
          description: String(form.get("description") ?? ""),
          price: Number(form.get("price") ?? 0),
          image_url: imageURLs[0] ?? "",
          image_urls: imageURLs
        })
      });
      formElement.reset();
      setNotice("出品しました");
      setSelectedListingID(null);
      setActiveTab("home");
      await loadListings();
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "出品に失敗しました");
    }
  }

  async function chooseProfileImage(event: React.ChangeEvent<HTMLInputElement>) {
    const image = event.target.files?.[0];
    if (!image) return;
    if (image.size > 560000) {
      setNotice("画像が大きすぎます。500KB以下の画像を選んでください。");
      return;
    }
    setEditImageURL(await fileToDataURL(image));
  }

  async function updateProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setNotice("");
    try {
      const updated = await request<User>("/api/me/profile", {
        method: "POST",
        body: JSON.stringify({
          name: editName,
          bio: editBio,
          profile_image_url: editImageURL
        })
      });
      setUser(updated);
      setProfileUser({
        id: updated.id,
        name: updated.name,
        profile_image_url: updated.profile_image_url,
        bio: updated.bio
      });
      await loadListings();
      await loadConversations();
      setProfileMode("view");
      setNotice("プロフィールを更新しました");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "プロフィール更新に失敗しました");
    }
  }

  async function generateDescription() {
    const title = (document.querySelector<HTMLInputElement>("[name='title']")?.value ?? "").trim();
    const condition = (document.querySelector<HTMLSelectElement>("[name='condition']")?.value ?? "").trim();
    const notes = (document.querySelector<HTMLTextAreaElement>("[name='description']")?.value ?? "").trim();
    const data = await request<{ description: string }>("/api/ai/description", {
      method: "POST",
      body: JSON.stringify({ title, condition, notes })
    });
    const area = document.querySelector<HTMLTextAreaElement>("[name='description']");
    if (area) area.value = data.description;
  }

  function openCheckout(item: Listing) {
    setSelectedListingID(item.id);
    setCheckoutListingID(item.id);
    setCheckoutMode("checkout");
    setProfileMode(null);
    setProfileUser(null);
    setPurchaseComplete(false);
    setNotice("");
  }

  async function cancelListing(item: Listing) {
    if (!window.confirm("この出品を取り消しますか？")) return;
    try {
      await request<Listing>(`/api/listings/${item.id}/cancel`, { method: "POST" });
      setSelectedListingID(null);
      setDetailSource("history");
      setNotice("出品を取り消しました");
      await loadListings();
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "出品取り消しに失敗しました");
    }
  }

  function formatAddress(address: ShippingAddress) {
    if (!address.lastName || !address.firstName || !address.postalCode || !address.prefecture || !address.city || !address.block) {
      return "";
    }
    return `${address.lastName} ${address.firstName} / 〒${address.postalCode} ${address.prefecture}${address.city}${address.block}${address.building ? ` ${address.building}` : ""}`;
  }

  function shippingAddressLine(address: ShippingAddress) {
    if (!address.postalCode || !address.prefecture || !address.city || !address.block) {
      return "";
    }
    return `〒${address.postalCode} ${address.prefecture}${address.city}${address.block}${address.building ? ` ${address.building}` : ""}`;
  }

  function recipientName(address: ShippingAddress) {
    return `${address.lastName} ${address.firstName}`.trim();
  }

  function backFromCheckout() {
    if (checkoutMode === "checkout") {
      setCheckoutMode(null);
      return;
    }
    setCheckoutMode("checkout");
  }

  async function confirmPurchase() {
    if (!checkoutListing) return;
    try {
      await request<PurchaseResult>(`/api/listings/${checkoutListing.id}/purchase`, {
        method: "POST",
        body: JSON.stringify({
          shipping_address: shippingAddressLine(shippingAddress),
          recipient_name: recipientName(shippingAddress)
        })
      });
      setCheckoutMode(null);
      setSelectedListingID(checkoutListing.id);
      setCheckoutListingID(null);
      setPurchaseComplete(true);
      await loadListings();
      await loadConversations();
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "購入に失敗しました");
    }
  }

  async function markPurchaseNotificationRead(notificationID: number) {
    await request<{ status: string }>(`/api/notifications/purchases/${notificationID}/read`, { method: "POST" });
    setPurchaseNotifications((current) => current.filter((notification) => notification.id !== notificationID));
  }

  async function openNotificationConversation(notification: PurchaseNotification) {
    await markPurchaseNotificationRead(notification.id);
    setCheckoutMode(null);
    setProfileMode(null);
    setActiveTab("messages");
    setActiveConversation(notification.conversation_id);
    await loadConversations();
    setMessages(await request<Message[]>(`/api/conversations/${notification.conversation_id}/messages`));
  }

  async function startConversation(listingID: number) {
    const conversation = await request<Conversation>("/api/conversations", {
      method: "POST",
      body: JSON.stringify({ listing_id: listingID })
    });
    setActiveTab("messages");
    setActiveConversation(conversation.id);
    await loadConversations();
    setMessages(await request<Message[]>(`/api/conversations/${conversation.id}/messages`));
  }

  async function sendMessage(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!activeConversation) return;
    const formElement = event.currentTarget;
    const form = new FormData(formElement);
    const attachment = form.get("attachment");
    try {
      if (attachment instanceof File && attachment.size > 560000) {
        setNotice("添付画像が大きすぎます。500KB以下の画像を選んでください。");
        return;
      }
      const attachmentURL = attachment instanceof File && attachment.size > 0 ? await fileToDataURL(attachment) : "";
      await request<Message>(`/api/conversations/${activeConversation}/messages`, {
        method: "POST",
        body: JSON.stringify({ body: String(form.get("body") ?? ""), attachment_url: attachmentURL })
      });
      formElement.reset();
      setMessages(await request<Message[]>(`/api/conversations/${activeConversation}/messages`));
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "メッセージ送信に失敗しました");
    }
  }

  function renderListingCard(item: Listing) {
    const bookmarked = bookmarks.includes(item.id);
    const images = listingImages(item);
    return (
      <article className="item-card" key={item.id}>
        <div className="item-image">
          {images[0] ? <img src={images[0]} alt={item.title} /> : <span>{item.title.slice(0, 1)}</span>}
        </div>
        <div className="item-body">
          <div className="item-head">
            <h3>{item.title}</h3>
            <span className={item.status}>{item.status === "sold" ? "Sold out" : "販売中"}</span>
          </div>
          <p>{item.description || "説明はまだありません"}</p>
          <div className="item-foot">
            <strong>¥{item.price.toLocaleString()}</strong>
            <span>{item.seller_name || `user #${item.seller_id}`}</span>
          </div>
          <div className="item-actions">
            <button disabled={item.status === "sold" || item.seller_id === user?.id} onClick={() => openCheckout(item)}>
              購入
            </button>
            <button className="secondary" disabled={item.seller_id === user?.id} onClick={() => startConversation(item.id)}>
              <MessageCircle size={16} /> DM
            </button>
            <button className={bookmarked ? "bookmark active" : "bookmark"} onClick={() => toggleBookmark(item.id)} aria-label="bookmark">
              <Bookmark size={16} />
            </button>
          </div>
        </div>
      </article>
    );
  }

  function renderListingSummaryCard(item: Listing, source: "home" | "history" = "home") {
    const images = listingImages(item);
    return (
      <button className="summary-card" key={item.id} onClick={() => openListingDetail(item, source)}>
        <div className="summary-image">
          {images[0] ? <img src={images[0]} alt={item.title} /> : <span>{item.title.slice(0, 1)}</span>}
          {item.status === "sold" && <span className="summary-sold-badge">Sold out</span>}
        </div>
        <div className="summary-info">
          <h3>{item.title}</h3>
          <strong>¥{item.price.toLocaleString()}</strong>
        </div>
      </button>
    );
  }

  function renderListingDetail(item: Listing) {
    const bookmarked = bookmarks.includes(item.id);
    const canCancel = detailSource === "history" && item.seller_id === user?.id && item.status === "available";
    const images = listingImages(item);
    const currentImage = images[selectedImageIndex] ?? "";
    return (
      <article className="detail-screen">
        <div className={canCancel ? "detail-topbar with-action" : "detail-topbar"}>
          <button className="icon-button" onClick={() => setSelectedListingID(null)} aria-label="back">
            ←
          </button>
          <strong>商品詳細</strong>
          {canCancel && (
            <button className="danger-small" onClick={() => cancelListing(item)}>
              出品取り消し
            </button>
          )}
        </div>
        <div className="detail-image">
          {currentImage ? <img src={currentImage} alt={`${item.title} ${selectedImageIndex + 1}`} /> : <span>{item.title.slice(0, 1)}</span>}
          {images.length > 1 && (
            <>
              <button
                className="image-nav prev"
                onClick={() => setSelectedImageIndex((current) => (current === 0 ? images.length - 1 : current - 1))}
                aria-label="前の画像"
              >
                ‹
              </button>
              <button
                className="image-nav next"
                onClick={() => setSelectedImageIndex((current) => (current + 1) % images.length)}
                aria-label="次の画像"
              >
                ›
              </button>
              <span className="image-count">
                {selectedImageIndex + 1} / {images.length}
              </span>
            </>
          )}
        </div>
        <div className="detail-body">
          <div className="detail-head">
            <div>
              <h2>{item.title}</h2>
              <strong>¥{item.price.toLocaleString()}</strong>
            </div>
            <span className={item.status}>{item.status === "sold" ? "Sold out" : "販売中"}</span>
          </div>
          <p>{item.description || "説明はまだありません"}</p>
          <button className="seller-line" onClick={() => openPublicProfile(item.seller_id)}>
            <UserRound size={16} />
            <span>{item.seller_name || `user #${item.seller_id}`}</span>
          </button>
          <div className="detail-actions">
            <button disabled={item.status === "sold" || item.seller_id === user?.id} onClick={() => openCheckout(item)}>
              購入
            </button>
            <button className="secondary" disabled={item.seller_id === user?.id} onClick={() => startConversation(item.id)}>
              <MessageCircle size={16} /> メッセージ
            </button>
            <button className={bookmarked ? "bookmark active" : "bookmark"} onClick={() => toggleBookmark(item.id)} aria-label="bookmark">
              <Bookmark size={16} />
            </button>
          </div>
        </div>
      </article>
    );
  }

  function renderCheckoutProduct(item: Listing) {
    const images = listingImages(item);
    return (
      <div className="checkout-product">
        <div className="checkout-thumb">
          {images[0] ? <img src={images[0]} alt={item.title} /> : <span>{item.title.slice(0, 1)}</span>}
        </div>
        <div>
          <h2>{item.title}</h2>
          <strong>¥{item.price.toLocaleString()}</strong>
          <p>送料込み</p>
        </div>
      </div>
    );
  }

  function renderCheckoutRow(label: string, value: string, onClick: () => void) {
    return (
      <button className="checkout-row" onClick={onClick}>
        <span>{label}</span>
        <span className={value ? "" : "muted"}>{value || "設定してください"}</span>
        <ChevronRight size={22} />
      </button>
    );
  }

  function renderCheckoutScreen(item: Listing) {
    const addressText = formatAddress(shippingAddress);
    return (
      <section className="purchase-screen">
        <div className="purchase-topbar">
          <button className="icon-button" onClick={backFromCheckout} aria-label="back">
            <ArrowLeft size={20} />
          </button>
          <strong>購入手続き</strong>
        </div>
        <div className="purchase-body">
          {renderCheckoutProduct(item)}
          <div className="checkout-card">
            {renderCheckoutRow("配送先", addressText, () => {
              setAddressDraft(shippingAddress);
              setCheckoutMode("address");
            })}
            {renderCheckoutRow("置き配", dropoffLocation, () => {
              setDropoffDraft(dropoffLocation || dropoffOptions[0]);
              setCheckoutMode("dropoff");
            })}
          </div>
          <div className="checkout-card">
            {renderCheckoutRow("支払い方法", paymentMethod, () => {
              setPaymentDraft(paymentMethod || paymentOptions[0]);
              setCheckoutMode("payment");
            })}
          </div>
          <div className="payment-summary">
            <div>
              <span>商品代金</span>
              <strong>¥{item.price.toLocaleString()}</strong>
            </div>
            <div>
              <span>支払い金額</span>
              <strong>¥{item.price.toLocaleString()}</strong>
            </div>
            <div>
              <span>支払い方法</span>
              <strong>{paymentMethod || "-"}</strong>
            </div>
          </div>
        </div>
        <div className="purchase-bottom">
          <button className="purchase-confirm-button" onClick={() => setCheckoutMode("confirm")}>
            購入確認へ
          </button>
        </div>
      </section>
    );
  }

  function renderAddressScreen() {
    const updateDraft = (key: keyof ShippingAddress, value: string) => setAddressDraft((current) => ({ ...current, [key]: value }));
    return (
      <section className="purchase-screen">
        <div className="purchase-topbar">
          <button className="icon-button" onClick={backFromCheckout} aria-label="back">
            <ArrowLeft size={20} />
          </button>
          <strong>住所の登録</strong>
        </div>
        <div className="purchase-form">
          <label>
            <span>姓（全角）</span>
            <input value={addressDraft.lastName} maxLength={15} placeholder="例）山田" onChange={(event) => updateDraft("lastName", event.target.value)} />
            <small>{addressDraft.lastName.length} / 15</small>
          </label>
          <label>
            <span>名（全角）</span>
            <input value={addressDraft.firstName} maxLength={15} placeholder="例）彩" onChange={(event) => updateDraft("firstName", event.target.value)} />
            <small>{addressDraft.firstName.length} / 15</small>
          </label>
          <label>
            <span>姓カナ（全角）</span>
            <input value={addressDraft.lastKana} maxLength={35} placeholder="例）ヤマダ" onChange={(event) => updateDraft("lastKana", event.target.value)} />
            <small>{addressDraft.lastKana.length} / 35</small>
          </label>
          <label>
            <span>名カナ（全角）</span>
            <input value={addressDraft.firstKana} maxLength={35} placeholder="例）アヤ" onChange={(event) => updateDraft("firstKana", event.target.value)} />
            <small>{addressDraft.firstKana.length} / 35</small>
          </label>
          <hr />
          <label>
            <span>郵便番号（数字）</span>
            <input value={addressDraft.postalCode} inputMode="numeric" placeholder="例）1234567" onChange={(event) => updateDraft("postalCode", event.target.value)} />
          </label>
          <label>
            <span>都道府県</span>
            <input value={addressDraft.prefecture} placeholder="例）東京都" onChange={(event) => updateDraft("prefecture", event.target.value)} />
          </label>
          <label>
            <span>市区町村</span>
            <input value={addressDraft.city} placeholder="例）渋谷区" onChange={(event) => updateDraft("city", event.target.value)} />
          </label>
          <label>
            <span>番地</span>
            <input value={addressDraft.block} placeholder="例）青山 1-1-1" onChange={(event) => updateDraft("block", event.target.value)} />
          </label>
          <label>
            <span>建物名 <em>任意</em></span>
            <input value={addressDraft.building} placeholder="例）柳ビル 103" onChange={(event) => updateDraft("building", event.target.value)} />
          </label>
          <label>
            <span>電話番号</span>
            <input value={addressDraft.phone} inputMode="tel" placeholder="例）09012345678" onChange={(event) => updateDraft("phone", event.target.value)} />
          </label>
        </div>
        <div className="purchase-bottom">
          <button
            className="purchase-confirm-button"
            onClick={() => {
              setShippingAddress(addressDraft);
              setCheckoutMode("checkout");
            }}
          >
            設定する
          </button>
        </div>
      </section>
    );
  }

  function renderDropoffScreen() {
    return (
      <section className="purchase-screen">
        <div className="purchase-topbar">
          <button className="icon-button" onClick={backFromCheckout} aria-label="back">
            <ArrowLeft size={20} />
          </button>
          <strong>指定した置き場所で受取</strong>
        </div>
        <div className="option-list">
          {dropoffOptions.map((option) => (
            <label className="option-row" key={option}>
              <input type="radio" name="dropoff" checked={dropoffDraft === option} onChange={() => setDropoffDraft(option)} />
              <span>{option}</span>
            </label>
          ))}
        </div>
        <div className="purchase-bottom">
          <button
            className="purchase-confirm-button"
            onClick={() => {
              setDropoffLocation(dropoffDraft);
              setCheckoutMode("checkout");
            }}
          >
            設定する
          </button>
        </div>
      </section>
    );
  }

  function renderPaymentScreen() {
    return (
      <section className="purchase-screen">
        <div className="purchase-topbar">
          <button className="icon-button" onClick={backFromCheckout} aria-label="back">
            <ArrowLeft size={20} />
          </button>
          <strong>支払い方法</strong>
        </div>
        <div className="option-list">
          <h2>クレジットでの支払い</h2>
          <label className="option-row">
            <input type="radio" name="payment" checked={paymentDraft === paymentOptions[0]} onChange={() => setPaymentDraft(paymentOptions[0])} />
            <CreditCard size={24} />
            <span>{paymentOptions[0]}</span>
          </label>
          <h2>その他の支払い方法</h2>
          {paymentOptions.slice(1).map((option) => (
            <label className="option-row" key={option}>
              <input type="radio" name="payment" checked={paymentDraft === option} onChange={() => setPaymentDraft(option)} />
              <span>{option}</span>
            </label>
          ))}
        </div>
        <div className="purchase-bottom">
          <button
            className="purchase-confirm-button"
            onClick={() => {
              setPaymentMethod(paymentDraft);
              setCheckoutMode("checkout");
            }}
          >
            設定する
          </button>
        </div>
      </section>
    );
  }

  function renderPurchaseConfirmScreen(item: Listing) {
    return (
      <section className="purchase-screen confirm">
        <div className="purchase-topbar">
          <button className="icon-button" onClick={backFromCheckout} aria-label="back">
            <ArrowLeft size={20} />
          </button>
          <strong>購入確認</strong>
        </div>
        <div className="purchase-body">
          <div className="confirm-table">
            <div>
              <span>商品代金</span>
              <strong>¥{item.price.toLocaleString()}</strong>
            </div>
            <div>
              <span>決済手数料</span>
              <strong>¥0</strong>
            </div>
            <div className="total">
              <span>支払い金額</span>
              <strong>¥{item.price.toLocaleString()}</strong>
            </div>
            <div>
              <span>支払い方法</span>
              <strong>{paymentMethod || "-"}</strong>
            </div>
            <div>
              <span>配送先</span>
              <strong>{formatAddress(shippingAddress) || "-"}</strong>
            </div>
            <div>
              <span>置き配</span>
              <strong>{dropoffLocation || "-"}</strong>
            </div>
          </div>
        </div>
        <div className="purchase-bottom">
          <button className="purchase-confirm-button" onClick={confirmPurchase}>
            購入する
          </button>
        </div>
      </section>
    );
  }

  function renderProfileAvatar(profile: PublicProfile | User) {
    return (
      <div className="profile-avatar">
        {profile.profile_image_url ? <img src={profile.profile_image_url} alt={profile.name} /> : <UserRound size={58} />}
      </div>
    );
  }

  function renderProfileScreen(profile: PublicProfile) {
    const isOwnProfile = profile.id === user?.id;
    return (
      <section className="profile-screen">
        <div className="profile-topbar">
          <button className="icon-button" onClick={() => setProfileMode(null)} aria-label="back">
            <ArrowLeft size={20} />
          </button>
          <strong>プロフィール</strong>
        </div>
        <div className="profile-body">
          {renderProfileAvatar(profile)}
          <h2>{profile.name}</h2>
          {isOwnProfile && (
            <button className="secondary edit-profile-button" onClick={openProfileEditor}>
              <Edit3 size={16} /> プロフィールを編集する
            </button>
          )}
          {profile.bio ? <p className="profile-bio">{profile.bio}</p> : <p className="profile-bio muted">自己紹介文は登録されていません</p>}
        </div>
      </section>
    );
  }

  function renderProfileEditScreen() {
    if (!user) return null;
    return (
      <form className="profile-edit-screen" onSubmit={updateProfile}>
        <div className="profile-topbar">
          <button className="icon-button" type="button" onClick={() => setProfileMode("view")} aria-label="back">
            <ArrowLeft size={20} />
          </button>
          <strong>プロフィール設定</strong>
        </div>
        <div className="profile-edit-body">
          <section className="profile-edit-section">
            <h2>プロフィール画像</h2>
            <div className="profile-image-row">
              {renderProfileAvatar({ ...user, name: editName, profile_image_url: editImageURL, bio: editBio })}
              <label className="image-change-button">
                <Camera size={18} />
                <span>画像を変更する</span>
                <input type="file" accept="image/*" hidden onChange={chooseProfileImage} />
              </label>
            </div>
          </section>

          <label className="profile-field">
            <span>ニックネーム</span>
            <input value={editName} maxLength={20} onChange={(event) => setEditName(event.target.value)} required />
            <small>{editName.length} / 20</small>
          </label>

          <label className="profile-field">
            <span>自己紹介文</span>
            <textarea
              value={editBio}
              maxLength={1000}
              rows={8}
              placeholder="例）ご覧いただきありがとうございます。お互いが気持ちの良いお取引をできるよう心がけています。よろしくお願いします。"
              onChange={(event) => setEditBio(event.target.value)}
            />
            <small>{editBio.length} / 1000</small>
          </label>

          <button className="profile-submit" type="submit">
            更新する
          </button>
        </div>
      </form>
    );
  }

  if (!user) {
    return (
      <main className="auth-screen">
        <section className="auth-card">
          <div className="auth-brand">
            <p className="eyebrow">Flea market website</p>
            <h1>Mr. Market</h1>
          </div>

          <form onSubmit={handleAuth} className="stack">
            <div className="segmented">
              <button type="button" className={authMode === "register" ? "active" : ""} onClick={() => setAuthMode("register")}>
                登録
              </button>
              <button type="button" className={authMode === "login" ? "active" : ""} onClick={() => setAuthMode("login")}>
                ログイン
              </button>
            </div>
            {authMode === "register" && <input name="name" placeholder="名前" required />}
            <input name="email" type="email" placeholder="メール" required />
            <input name="password" type="password" placeholder="パスワード（8文字以上）" required />
            <button type="submit">{authMode === "register" ? "登録してホームへ" : "ログインしてホームへ"}</button>
          </form>
        </section>
      </main>
    );
  }

  return (
    <main className="app-shell">
      {!checkoutMode && (
        <header className="topbar">
          <div>
            <p className="eyebrow">Flea market website</p>
            <h1>Mr. Market</h1>
          </div>
          <div className="topbar-actions">
            <button className="account-pill" onClick={openOwnProfile}>
              <UserRound size={16} />
              <span>{user.name}</span>
            </button>
            <button className="ghost" onClick={refreshAppData}>
              <RefreshCw size={16} /> 更新
            </button>
            <button className="ghost" onClick={logout}>
              <LogOut size={16} /> ログアウト
            </button>
          </div>
        </header>
      )}

      {checkoutMode === "checkout" && checkoutListing && renderCheckoutScreen(checkoutListing)}

      {checkoutMode === "address" && renderAddressScreen()}

      {checkoutMode === "dropoff" && renderDropoffScreen()}

      {checkoutMode === "payment" && renderPaymentScreen()}

      {checkoutMode === "confirm" && checkoutListing && renderPurchaseConfirmScreen(checkoutListing)}

      {!checkoutMode && profileMode === "view" && profileUser && renderProfileScreen(profileUser)}

      {!checkoutMode && profileMode === "edit" && renderProfileEditScreen()}

      {!checkoutMode && !profileMode && activeTab === "home" && (
        <section className="tab-panel">
          {selectedListing ? (
            renderListingDetail(selectedListing)
          ) : (
            <>
              <form
                className="searchbar"
                onSubmit={(event) => {
                  event.preventDefault();
                  setAppliedQuery(searchDraft);
                }}
              >
                <button className="search-button" aria-label="検索">
                  <Search size={18} />
                </button>
                <input value={searchDraft} onChange={(event) => setSearchDraft(event.target.value)} placeholder="商品名・説明・出品者で検索" />
              </form>

              <div className="section-heading">
                <ShoppingBag size={20} />
                <h2>新着</h2>
              </div>
              <div className="summary-grid">{filteredListings.map((item) => renderListingSummaryCard(item, "home"))}</div>
            </>
          )}
        </section>
      )}

      {!checkoutMode && !profileMode && activeTab === "sell" && (
        <section className="panel tab-panel">
          <div className="panel-title">
            <Plus size={18} />
            <h2>出品</h2>
          </div>
          <form onSubmit={createListing} className="listing-form">
            <input name="title" placeholder="例: 経済学の教科書" required />
            <input name="price" type="number" min="1" placeholder="価格（円）" required />
            <input name="images" type="file" accept="image/*" multiple />
            <small className="form-hint">画像は5枚まで。大きい写真は自動で軽くして保存します。</small>
            <select name="condition" required defaultValue="">
              <option value="" disabled>
                状態を選択
              </option>
              {conditionOptions.map((condition) => (
                <option value={condition} key={condition}>
                  {condition}
                </option>
              ))}
            </select>
            <textarea name="description" placeholder="メモ、補足" rows={6} />
            <div className="form-actions">
              <button type="button" className="secondary" onClick={generateDescription}>
                <Bot size={16} /> AI説明生成
              </button>
              <button type="submit">出品する</button>
            </div>
          </form>
        </section>
      )}

      {!checkoutMode && !profileMode && activeTab === "messages" && (
        <section className="tab-panel">
          {!activeConversation && (
            <>
              <div className="section-heading">
                <MessageCircle size={20} />
                <h2>メッセージ</h2>
              </div>
              <div className="chat-list">
                {conversations.length === 0 && <p className="empty-state">チャットはまだありません。</p>}
                {conversations.map((conversation) => (
                  <button
                    key={conversation.id}
                    className="chat-row"
                    onClick={async () => {
                      setActiveConversation(conversation.id);
                      setMessages(await request<Message[]>(`/api/conversations/${conversation.id}/messages`));
                    }}
                  >
                    <span className="avatar">{conversation.other_user_name.slice(0, 1)}</span>
                    <span>
                      <strong>{conversation.other_user_name}</strong>
                      <small>{conversation.title}</small>
                    </span>
                  </button>
                ))}
              </div>
            </>
          )}

          {activeConversation && (
            <div className="chat-screen">
              <div className="chat-header">
                <button className="icon-button" onClick={() => setActiveConversation(null)} aria-label="back">
                  ←
                </button>
                <button
                  className="chat-profile-button"
                  disabled={!activeConversationDetail}
                  onClick={() => activeConversationDetail && openPublicProfile(activeConversationDetail.other_user_id)}
                >
                  {activeConversationDetail?.other_user_name ?? "チャット"}
                </button>
              </div>

              <div className="message-thread">
                {messages.map((message) => (
                  <div key={message.id} className={message.sender_id === user.id ? "message-bubble mine" : "message-bubble"}>
                    {message.attachment_url && <img src={message.attachment_url} alt="添付画像" />}
                    {message.body && <p>{message.body}</p>}
                  </div>
                ))}
              </div>

              <form onSubmit={sendMessage} className="chat-composer">
                <input id="message-attachment" name="attachment" type="file" accept="image/*" hidden />
                <button type="button" className="icon-button" onClick={() => document.querySelector<HTMLInputElement>("#message-attachment")?.click()} aria-label="attach">
                  <Plus size={20} />
                </button>
                <input name="body" placeholder="メッセージを入力" />
                <button>
                  <Send size={16} /> 送信
                </button>
              </form>
            </div>
          )}
        </section>
      )}

      {!checkoutMode && !profileMode && activeTab === "history" && (
        <section className="tab-panel">
          {selectedListing && detailSource === "history" ? (
            renderListingDetail(selectedListing)
          ) : (
            <>
              <div className="section-heading">
                <Clock3 size={20} />
                <h2>履歴</h2>
              </div>
              <section className="history-group">
                <h3>ブックマーク</h3>
                {bookmarkedListings.length > 0 ? (
                  <div className="summary-grid">{bookmarkedListings.map((item) => renderListingSummaryCard(item, "history"))}</div>
                ) : (
                  <p className="empty-state">ブックマークした商品はまだありません。</p>
                )}
              </section>
              <section className="history-group">
                <h3>購入済み</h3>
                {purchasedListings.length > 0 ? (
                  <div className="summary-grid">{purchasedListings.map((item) => renderListingSummaryCard(item, "history"))}</div>
                ) : (
                  <p className="empty-state">購入済みの商品はまだありません。</p>
                )}
              </section>
              <section className="history-group">
                <h3>出品中</h3>
                {sellingListings.length > 0 ? (
                  <div className="summary-grid">{sellingListings.map((item) => renderListingSummaryCard(item, "history"))}</div>
                ) : (
                  <p className="empty-state">出品中の商品はまだありません。</p>
                )}
              </section>
              <section className="history-group">
                <h3>売却済み</h3>
                {soldListings.length > 0 ? (
                  <div className="summary-grid">{soldListings.map((item) => renderListingSummaryCard(item, "history"))}</div>
                ) : (
                  <p className="empty-state">売却済みの商品はまだありません。</p>
                )}
              </section>
            </>
          )}
        </section>
      )}

      {purchaseComplete && (
        <div className="modal-backdrop">
          <div className="complete-modal" role="dialog" aria-modal="true">
            <h2>購入完了しました</h2>
            <button onClick={() => setPurchaseComplete(false)}>閉じる</button>
          </div>
        </div>
      )}

      {purchaseNotifications[0] && (
        <div className="modal-backdrop">
          <div className="complete-modal purchase-notification-modal" role="dialog" aria-modal="true">
            <h2>商品が購入されました</h2>
            <p>
              以下の商品が購入されました。
              <strong>{purchaseNotifications[0].title}</strong>
            </p>
            <div className="modal-actions">
              <button onClick={() => openNotificationConversation(purchaseNotifications[0])}>メッセージへ</button>
              <button className="secondary" onClick={() => markPurchaseNotificationRead(purchaseNotifications[0].id)}>
                閉じる
              </button>
            </div>
          </div>
        </div>
      )}

      {!checkoutMode && (
      <nav className="bottom-tabs" aria-label="main navigation">
        <button className={activeTab === "home" ? "active" : ""} onClick={() => switchTab("home")}>
          <Home size={20} />
          <span>ホーム</span>
        </button>
        <button className={activeTab === "sell" ? "active" : ""} onClick={() => switchTab("sell")}>
          <Plus size={20} />
          <span>出品</span>
        </button>
        <button className={activeTab === "messages" ? "active" : ""} onClick={() => switchTab("messages")}>
          <MessageCircle size={20} />
          <span>メッセージ</span>
        </button>
        <button className={activeTab === "history" ? "active" : ""} onClick={() => switchTab("history")}>
          <Clock3 size={20} />
          <span>履歴</span>
        </button>
      </nav>
      )}
    </main>
  );
}

createRoot(document.getElementById("root")!).render(<App />);
