import { redirect } from '@sveltejs/kit';

export const load = ({ params }: { params: { siteID: string } }) => {
	throw redirect(307, `/sites/${params.siteID}/store/products`);
};
